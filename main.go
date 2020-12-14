package main

import (
	"context"
	"flag"
	"time"

	"kube-sqs-autoscaler/scale"
	kubesqs "kube-sqs-autoscaler/sqs"

	log "github.com/sirupsen/logrus"
)

var (
	pollInterval        time.Duration
	scaleDownCoolPeriod time.Duration
	scaleUpCoolPeriod   time.Duration
	scaleUpMessages     int
	scaleDownMessages   int
	scaleUpPods         int
	scaleDownPods       int
	maxPods             int
	minPods             int
	awsRegion           string

	sqsQueueUrl              string
	kubernetesDeploymentName string
	kubernetesNamespace      string
)

func Run(p *scale.PodAutoScaler, sqs *kubesqs.SqsClient) {
	ctx := context.Background()
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
						log.Info("Waiting for cool down, skipping scale up ")
						continue
					}

					if err := p.ScaleUp(ctx); err != nil {
						log.Errorf("Failed scaling up: %v", err)
						continue
					}

					lastScaleUpTime = time.Now()
				}

				if numMessages <= scaleDownMessages {
					if lastScaleDownTime.Add(scaleDownCoolPeriod).After(time.Now()) {
						log.Info("Waiting for cool down, skipping scale down")
						continue
					}

					if err := p.ScaleDown(ctx); err != nil {
						log.Errorf("Failed scaling down: %v", err)
						continue
					}

					lastScaleDownTime = time.Now()
				}
			}
		}
	}

}

func main() {
	flag.DurationVar(&pollInterval, "poll-period", 5*time.Second, "The interval in seconds for checking if scaling is required")
	flag.DurationVar(&scaleDownCoolPeriod, "scale-down-cool-down", 30*time.Second, "The cool down period for scaling down")
	flag.DurationVar(&scaleUpCoolPeriod, "scale-up-cool-down", 10*time.Second, "The cool down period for scaling up")
	flag.IntVar(&scaleUpMessages, "scale-up-messages", 100, "Number of sqs messages queued up required for scaling up")
	flag.IntVar(&scaleDownMessages, "scale-down-messages", 10, "Number of messages required to scaling down")
	flag.IntVar(&scaleUpPods, "scale-up-pods", 1, "Number of Pod in scaling up")
	flag.IntVar(&scaleDownPods, "scale-down-pods", 1, "Number of Pod in scaling down")
	flag.IntVar(&maxPods, "max-pods", 5, "Max pods that kube-sqs-autoscaler can scale")
	flag.IntVar(&minPods, "min-pods", 1, "Min pods that kube-sqs-autoscaler can scale")
	flag.StringVar(&awsRegion, "aws-region", "", "Your AWS region")

	flag.StringVar(&sqsQueueUrl, "sqs-queue-url", "", "The sqs queue url")
	flag.StringVar(&kubernetesDeploymentName, "kubernetes-deployment", "", "Kubernetes Deployment to scale. This field is required")
	flag.StringVar(&kubernetesNamespace, "kubernetes-namespace", "default", "The namespace your deployment is running in")

	flag.Parse()

	p := scale.NewPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, scaleUpPods, scaleDownPods)
	sqs := kubesqs.NewSqsClient(sqsQueueUrl, awsRegion)

	log.Info("Starting kube-sqs-autoscaler")
	Run(p, sqs)
}
