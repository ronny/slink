package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/ronny/slink"
	"github.com/ronny/slink/storage"
	_ "go.uber.org/automaxprocs"
)

const BootTimeout = 5 * time.Second

func main() {
	ctx, cancelCtx := context.WithTimeout(context.Background(), BootTimeout)
	defer cancelCtx()

	slinkOptions := make([]func(*slink.Slink), 0)

	// TODO: use flag to build slink options, including to use ddblocal
	{
		ddbCfg, err := config.LoadDefaultConfig(ctx,
			config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(
					func(service, region string, options ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{URL: "http://localhost:8000"}, nil
					},
				),
			),
			// TODO: remove hardcoded keys, just use env vars
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "slink",
					SecretAccessKey: "slink",
					SessionToken:    "slink",
				},
			}),
		)
		if err != nil {
			log.Fatalf("(aws)config.LoadDefaultConfig: %s", err.Error())
		}

		ddblocal, err := storage.NewDynamoDBStorage(ctx, storage.WithDynamoDBConfig(ddbCfg))
		if err != nil {
			log.Fatalf("storage.NewDynamoDBStorage: %s", err.Error())
		}
		slinkOptions = append(slinkOptions, slink.WithStorage(ddblocal))
	}

	listenAddr := ":8080"

	log.Printf("public server listening on %s", listenAddr)
	s, err := NewPublicServer(ctx,
		WithListenAddr(listenAddr),
		WithSlinkOptions(slinkOptions...),
	)
	if err != nil {
		log.Fatalf("NewPublicServer: %s", err.Error())
	}

	err = s.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe: %s", err.Error())
	}
}
