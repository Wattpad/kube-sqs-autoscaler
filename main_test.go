package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"kube-sqs-autoscaler/scale"
	mainsqs "kube-sqs-autoscaler/sqs"
)

func TestRunReachMinReplicas(t *testing.T) {
	ctx := context.Background()
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("10")}
	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(minPods), *deployment.Spec.Replicas, "Number of replicas should be the min")
}

func TestRunReachMaxReplicas(t *testing.T) {
	ctx := context.Background()
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("100")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(maxPods), *deployment.Spec.Replicas, "Number of replicas should be the max")
}

func TestRunScaleUpCoolDown(t *testing.T) {
	ctx := context.Background()
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("100")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(4), *deployment.Spec.Replicas, "Number of replicas should be 4 if cool down for scaling up was obeyed")
}

func TestRunScaleDownCoolDown(t *testing.T) {
	ctx := context.Background()
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("10")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(2), *deployment.Spec.Replicas, "Number of replicas should be 2 if cool down for scaling down was obeyed")
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int, init int) *scale.PodAutoScaler {
	initialReplicas := int32(init)
	mock := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "deploy",
			Namespace:   "namespace",
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	}, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "deploy-no-scale",
			Namespace:   "namespace",
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	})
	return &scale.PodAutoScaler{
		Client:     mock.AppsV1().Deployments(kubernetesNamespace),
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}

type MockSQS struct {
	QueueAttributes *sqs.GetQueueAttributesOutput
}

func (m *MockSQS) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	return m.QueueAttributes, nil
}

func (m *MockSQS) SetQueueAttributes(input *sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	m.QueueAttributes = &sqs.GetQueueAttributesOutput{
		Attributes: input.Attributes,
	}
	return &sqs.SetQueueAttributesOutput{}, nil
}

func NewMockSqsClient() *mainsqs.SqsClient {
	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("50")}

	return &mainsqs.SqsClient{
		Client: &MockSQS{
			QueueAttributes: &sqs.GetQueueAttributesOutput{
				Attributes: Attributes,
			},
		},
		QueueUrl: "example.com",
	}
}
