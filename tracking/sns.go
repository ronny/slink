package tracking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type SNSTracker struct {
	topicARN string
	client   SNSPublisher
}

var _ Tracker = (*SNSTracker)(nil)

func (t *SNSTracker) TrackShortLinkLookupRequest(ctx context.Context, payload *ShortLinkLookupPayload) error {
	messageJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(t.topicARN),
		Message:  aws.String(string(messageJSON)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"requestHost": {
				DataType:    aws.String("String"),
				StringValue: aws.String(payload.RequestHost),
			},
			"shortLinkId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(payload.ShortLinkID),
			},
		},
	}

	_, err = t.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("sns.Publish: %w", err)
	}
	return nil
}

type SNSPublisher interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func NewSNSTracker(ctx context.Context, topicARN string, options ...func(*SNSTracker)) (*SNSTracker, error) {
	if topicARN == "" {
		return nil, errors.New("missing topicARN")
	}

	t := &SNSTracker{
		topicARN: topicARN,
	}

	for _, option := range options {
		option(t)
	}

	if t.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("awsConfig.LoadDefaultConfig: %w", err)
		}
		t.client = sns.NewFromConfig(cfg)
	}

	return t, nil
}

func WithSNSClient(client SNSPublisher) func(*SNSTracker) {
	return func(s *SNSTracker) {
		s.client = client
	}
}
