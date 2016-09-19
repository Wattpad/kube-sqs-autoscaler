package main

import (
	"flag"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/Wattpad/kube-sqs-autoscaler/scale"
	"github.com/Wattpad/kube-sqs-autoscaler/sqs"
)

var (
	pollInterval        = *flag.Duration("poll-period", 5*time.Second, "The interval in seconds for checking if scaling is required")
	scaleDownCoolPeriod = *flag.Duration("scale-down-cool-down", 30*time.Second, "The cool down period for scaling down")
	scaleUpCoolPeriod   = *flag.Duration("scale-up-cool-down", 10*time.Second, "The cool down period for scaling up")
	scaleUpMessages     = *flag.Int("scale-up-messages", 100, "Number of sqs messages queued up required for scaling up")
	scaleDownMessages   = *flag.Int("scale-down-messages", 10, "Number of messages required to scale down")
	maxPods             = *flag.Int("max-pods", 5, "Max pods that kube-sqs-autoscaler can scale")
	minPods             = *flag.Int("min-pods", 1, "Min pods that kube-sqs-autoscaler can scale")
	awsRegion           = *flag.String("aws-region", "us-east-1", "Your AWS region")

	sqsQueueUrl              = *flag.String("sqs-queue-url", "", "The sqs queue url")
	kubernetesDeploymentName = *flag.String("kubernetes-deployment", "", "Kubernetes Deployment to scale. This field is required")
	kubernetesNamespace      = *flag.String("kubernetes-namespace", "default", "The namespace your deployment is running in")
)

func Run(p *scale.PodAutoScaler, sqs *sqs.SqsClient) {
	lastScaleUpTime := time.Now()
	lastScaleDownTime := time.Now()

	for {
		select {
		case <-time.After(pollInterval):
			{
				numMessages, err := sqs.NumMessages()
				if err != nil {
					log.Errorf("Failed to get SQS messages: %v", err)
					continue
				}

				if numMessages >= scaleUpMessages {
					if lastScaleUpTime.Add(scaleUpCoolPeriod).After(time.Now()) {
						continue
					}

					if err := p.ScaleUp(); err != nil {
						log.Errorf("Failed scaling up: %v", err)
						continue
					}

					lastScaleUpTime = time.Now()
					continue
				}

				if numMessages <= scaleDownMessages {
					if lastScaleDownTime.Add(scaleDownCoolPeriod).After(time.Now()) {
						continue
					}

					if err := p.ScaleDown(); err != nil {
						log.Errorf("Failed scaling down: %v", err)
						continue
					}

					lastScaleDownTime = time.Now()
					continue
				}
			}
		}
	}

}

func main() {
	flag.Parse()

	p := scale.NewPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	sqs := sqs.NewSqsClient(sqsQueueUrl, awsRegion)

	Run(p, sqs)
}
