package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/peterbourgon/ff/v3"
	"github.com/ronny/slink"
	"github.com/ronny/slink/debug"
	"github.com/ronny/slink/ids"
	"github.com/ronny/slink/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/automaxprocs/maxprocs"
)

const BootTimeout = 5 * time.Second

func main() {
	maxprocs.Set(maxprocs.Logger(log.Info().Msgf))

	fs := flag.NewFlagSet("slink-admin-server", flag.ExitOnError)

	var (
		listenAddr          = fs.String("listen-addr", ":9090", "the host:port address where the ddmin server should listen to")
		length              = fs.Int("length", 10, "the length of the ID to generate, see https://zelark.github.io/nano-id-cc/")
		chars               = fs.String("chars", ids.NanoIDDefaultCharacters, "the allowed characters used for generating IDs")
		denylistFilename    = fs.String("denylist", "", "custom denylist.txt file to use for checking generated IDs (optional)")
		denylistMaxAttempts = fs.Int("denylist-max-attempts", 10, "max number of attempts generating an ID and comparing against denylist before giving up")
		dynamodbTableName   = fs.String("dynamodb-tablename", storage.DynamoDBDefaultTableName, "the dynamodb table name")
		dynamodbRegion      = fs.String("dynamodb-region", storage.DynamoDBDefaultRegion, "the dynamodb region")
		dynamodbEndpoint    = fs.String("dynamodb-endpoint", "", "custom dynamodb endpoint URL to use, e.g. `http://localhost:8000` for dynamodb-local (optional)")
		awsAccessKeyID      = fs.String("aws-access-key-id", "", "override AWS_ACCESS_KEY_ID used for dynamodb, only for local development with dynamodb-local, useful for namespacing a shared dynamodb-local (optional)")
		debugListenAddr     = fs.String("debug-listen-addr", "", "the host:port address where the debug server should listen to (optional, only launched when specified)")
		prettyLog           = fs.Bool("pretty-log", false, "whether to enable logs pretty-printing (inefficient), otherwise json")
		logLevel            = fs.String("log-level", "info", "set the minimum log level")
		_                   = fs.String("config", "", "config file (optional)")
	)

	err := ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarNoPrefix(),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.JSONParser),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("ff.Parse")
	}

	{
		level, err := zerolog.ParseLevel(*logLevel)
		if err != nil {
			log.Fatal().Err(err).Msg("zerolog.ParseLevel")
		}
		zerolog.SetGlobalLevel(level)

		if *prettyLog {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), BootTimeout)
	defer cancelCtx()

	slinkOptions := make([]func(*slink.Slink), 0)

	// Storage / DynamoDB
	{
		awsConfigOpts := []func(*config.LoadOptions) error{
			config.WithRegion(*dynamodbRegion),
		}

		if *dynamodbEndpoint != "" {
			awsConfigOpts = append(awsConfigOpts, config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(
					func(service, region string, options ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{URL: *dynamodbEndpoint}, nil
					},
				),
			))
		}
		if *awsAccessKeyID != "" {
			awsConfigOpts = append(awsConfigOpts, config.WithCredentialsProvider(
				credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID:     *awsAccessKeyID, // the value is used by dynamodb-local for namespacing
						SecretAccessKey: *awsAccessKeyID, // the value doesn't matter, just needs to exist for dynamodb-local
						SessionToken:    *awsAccessKeyID, // the value doesn't matter, just needs to exist for dynamodb-local
					},
				},
			))
		}
		ddbCfg, err := config.LoadDefaultConfig(ctx, awsConfigOpts...)
		if err != nil {
			log.Fatal().Err(err).Msg("(aws)config.LoadDefaultConfig")
		}

		ddblocal, err := storage.NewDynamoDBStorage(ctx,
			storage.WithDynamoDBConfig(ddbCfg),
			storage.WithDynamoDBTableName(*dynamodbTableName),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("storage.NewDynamoDBStorage")
		}
		slinkOptions = append(slinkOptions, slink.WithStorage(ddblocal))
	}

	// ID Generator
	{
		nanoidOpts := []func(*ids.NanoIDGenerator){
			ids.WithNanoIDLength(*length),
			ids.WithNanoIDCustomASCII(*chars),
			ids.WithNanoIDMaxAttempts(*denylistMaxAttempts),
		}

		if *denylistFilename != "" {
			denylist, err := ids.LoadDenylist(*denylistFilename)
			if err != nil {
				log.Fatal().Err(err).Str("denylistFilename", *denylistFilename).Msg("LoadDenylist")
			}
			nanoidOpts = append(nanoidOpts, ids.WithNanoIDDenylist(denylist))
		}

		nanoidGenerator, err := ids.NewNanoIDGenerator(nanoidOpts...)
		if err != nil {
			log.Fatal().Err(err).Msg("NewNanoIDGenerator")
		}
		slinkOptions = append(slinkOptions, slink.WithIDGenerator(nanoidGenerator))
	}

	log.Info().
		Str("dynamodbEndpoint", *dynamodbEndpoint).
		Str("awsAccessKeyID", *awsAccessKeyID).
		Int("length", *length).
		Str("chars", *chars).
		Int("denylistMaxAttempts", *denylistMaxAttempts).
		Str("denylistFilename", *denylistFilename).
		Str("debugListenAddr", *debugListenAddr).
		Msg("slink-admin-server flags")

	adminServer, err := NewAdminServer(ctx,
		WithListenAddr(*listenAddr),
		WithSlinkOptions(slinkOptions...),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("NewAdminServer")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Debug().Str("addr", *listenAddr).Msg("starting admin server...")
	go func() {
		err = adminServer.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("admin server ListenAndServe returned an unexpected error")
		}
		log.Info().Msg("admin server closed")
	}()
	log.Info().Str("addr", *listenAddr).Msg("admin server started")

	var debugServer *debug.DebugServer
	if *debugListenAddr != "" {
		debugServer, err := debug.NewDebugServer(*debugListenAddr)
		if err != nil {
			log.Fatal().Err(err).Msg("debug.NewDebugServer")
		}
		go func() {
			err = debugServer.ListenAndServe()
			if err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("debug server ListenAndServe returned an unexpected error")
			}
			log.Info().Msg("debug server closed")
		}()
		log.Info().Str("addr", *debugListenAddr).Msg("debug server started")
	}

	sig := <-sigChan
	log.Info().Msgf("received signal %v, shutting down admin server gracefully...", sig)

	gracefulShutdownCtx, gracefulShutdownCancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
	defer gracefulShutdownCancelCtx()

	if debugServer != nil {
		go debugServer.Shutdown(gracefulShutdownCtx)
	}

	err = adminServer.Shutdown(gracefulShutdownCtx)
	if err != nil {
		log.Fatal().Err(err).Msg("admin server shutdown failed")
	}

	log.Info().Msg("admin server gracefully shut down, bye")
}
