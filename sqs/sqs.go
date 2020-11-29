package sqs

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/pkg/errors"
)

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	// only implemented on unit tests
	SetQueueAttributes(*sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error)
}

type SqsClient struct {
	Client   SQS
	QueueUrl string
}

func NewSqsClient(queue string, region string) *SqsClient {
	svc := sqs.New(session.Must(session.NewSession()), aws.NewConfig().WithRegion(region))
	return &SqsClient{
		svc,
		queue,
	}
}

func (s *SqsClient) NumMessages() (int, error) {
	params := &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{
			aws.String("ApproximateNumberOfMessages"),
			aws.String("ApproximateNumberOfMessagesDelayed"),
			aws.String("ApproximateNumberOfMessagesNotVisible")},
		QueueUrl: aws.String(s.QueueUrl),
	}

	out, err := s.Client.GetQueueAttributes(params)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get messages in SQS")
	}

	approximateNumberOfMessages, err := strconv.Atoi(*out.Attributes["ApproximateNumberOfMessages"])
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get number of messages in queue")
	}

	approximateNumberOfMessagesDelayed, err := strconv.Atoi(*out.Attributes["ApproximateNumberOfMessagesDelayed"])
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get number of messages in queue")
	}

	approximateNumberOfMessagesNotVisible, err := strconv.Atoi(*out.Attributes["ApproximateNumberOfMessagesNotVisible"])
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get number of messages in queue")
	}

	messages := approximateNumberOfMessages + approximateNumberOfMessagesDelayed + approximateNumberOfMessagesNotVisible

	return messages, nil
}
