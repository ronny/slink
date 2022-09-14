package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ronny/slink/models"
)

const (
	DynamoDBDefaultTableName = "slink"
	DynamoDBDefaultGSI1Name  = "GSI1"
	DynamoDBDefaultRegion    = "us-east-1"
)

type DynamoDBStorage struct {
	tableName string
	gsi1Name  string
	awsConfig *aws.Config
	client    DynamoDBClient
}

func (d *DynamoDBStorage) Store(ctx context.Context, shortLink *models.ShortLink) error {
	item := &ddbShortLinkItem{
		ShortLink: shortLink,
		Type:      "ShortLink",
		PK:        shortLink.ID,
		SK:        shortLink.ID,
		GSI1PK:    shortLink.LinkURL,
		GSI1SK:    shortLink.ID,
	}

	avItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("ddbAV.MarshalMap: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      avItem,
	})
	if err != nil {
		return fmt.Errorf("ddb.PutItem: %w: %v", err, avItem)
	}
	return nil
}

func (d *DynamoDBStorage) GetByID(ctx context.Context, shortLinkID string) (*models.ShortLink, error) {
	output, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: shortLinkID},
			"sk": &types.AttributeValueMemberS{Value: shortLinkID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ddb.GetItem: %w", err)
	}

	var av *ddbShortLinkItem
	err = attributevalue.UnmarshalMap(output.Item, &av)
	if err != nil {
		return nil, fmt.Errorf("ddbAV.UnmarshalMap: %s", err)
	}
	return av.ShortLink, nil
}

func (d *DynamoDBStorage) GetByURL(ctx context.Context, linkURL string) ([]*models.ShortLink, error) {
	output, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		IndexName:              aws.String(d.gsi1Name),
		KeyConditionExpression: aws.String("#gsi1pk = :gsi1pk"),
		ExpressionAttributeNames: map[string]string{
			"#gsi1pk": "gsi1pk",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: linkURL},
		},
		// TODO: ExclusiveStartKey
		// TODO: Limit
	})
	if err != nil {
		return nil, fmt.Errorf("ddb.Query: %w", err)
	}

	// TODO: handle pagination

	result := make([]*models.ShortLink, 0)
	for _, item := range output.Items {
		var av *ddbShortLinkItem
		err = attributevalue.UnmarshalMap(item, &av)
		if err != nil {
			return nil, fmt.Errorf("ddbAV.UnmarshalMap: %s", err)
		}
		result = append(result, av.ShortLink)
	}
	return result, nil
}

type ddbShortLinkItem struct {
	*models.ShortLink

	Type   string `dynamodbav:"_type"`
	PK     string `dynamodbav:"pk"`
	SK     string `dynamodbav:"sk"`
	GSI1PK string `dynamodbav:"gsi1pk"`
	GSI1SK string `dynamodbav:"gsi1sk"`
}

func NewDynamoDBStorage(ctx context.Context, options ...func(*DynamoDBStorage)) (*DynamoDBStorage, error) {
	s := &DynamoDBStorage{}

	for _, option := range options {
		option(s)
	}

	if s.tableName == "" {
		s.tableName = DynamoDBDefaultTableName
	}

	if s.gsi1Name == "" {
		s.gsi1Name = DynamoDBDefaultGSI1Name
	}

	if s.awsConfig == nil {
		var err error
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("awsConfig.LoadDefaultConfig: %w", err)
		}
		s.awsConfig = &cfg
	}

	if s.client == nil {
		s.client = dynamodb.NewFromConfig(*s.awsConfig)
	}

	err := s.ensureTable(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *DynamoDBStorage) ensureTable(ctx context.Context) error {
	var err error
	backoffSchedule := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		10 * time.Second,
	}
	for _, backoff := range backoffSchedule {
		var output *dynamodb.DescribeTableOutput
		output, err = s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(s.tableName),
		})
		if err == nil && output != nil && output.Table != nil && output.Table.TableArn != nil {
			log.Printf("Table ARN: %s", *output.Table.TableArn)
			break
		}

		if err != nil {
			var rnfe *types.ResourceNotFoundException
			if errors.As(err, &rnfe) {
				log.Println("table missing")
				{
					err := s.createTable(ctx)
					if err != nil {
						return err
					}
				}
			} else {
				return fmt.Errorf("client.DescribeTable: %w", err)
			}
		}
		log.Printf("waiting for %s...", backoff)
		time.Sleep(backoff)
	}
	if err != nil {
		return fmt.Errorf("describe/create table: %w", err)
	}
	return nil
}

func (s *DynamoDBStorage) createTable(ctx context.Context) error {
	if s.client == nil {
		return errors.New("BUG: createTable called before s.client is set")
	}
	if s.tableName == "" {
		return errors.New("BUG: createTable called before s.tableName is set")
	}

	log.Println("creating table...")

	_, err := s.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String(s.tableName),
		BillingMode: types.BillingModePayPerRequest,
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("sk"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("gsi1pk"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("gsi1sk"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("gsi1pk"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("gsi1sk"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	})
	return err
}

func WithDynamoDBTableName(tableName string) func(*DynamoDBStorage) {
	return func(s *DynamoDBStorage) {
		s.tableName = tableName
	}
}

func WithDynamoDBGSI1Name(gsi1Name string) func(*DynamoDBStorage) {
	return func(s *DynamoDBStorage) {
		s.gsi1Name = gsi1Name
	}
}

func WithDynamoDBConfig(cfg aws.Config) func(*DynamoDBStorage) {
	return func(s *DynamoDBStorage) {
		s.awsConfig = &cfg
	}
}

func WithDynamoDBClient(client DynamoDBClient) func(*DynamoDBStorage) {
	return func(s *DynamoDBStorage) {
		s.client = client
	}
}

type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, options ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, options ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
}