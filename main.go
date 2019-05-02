package main

import (
	"flag"
	"time"
	"math"

	log "github.com/Sirupsen/logrus"

	"github.com/ddonaghy-r7/kube-sqs-autoscaler/scale"
	"github.com/ddonaghy-r7/kube-sqs-autoscaler/sqs"
)

var (
	pollInterval        time.Duration
	scaleDownCoolPeriod time.Duration
	scaleUpCoolPeriod   time.Duration
	scaleUpMessages     int
	scaleDownMessages   int
	maxPods             int
	minPods             int
	scaleByRatio        bool
	queuePerPodRatio    int
	awsRegion           string

	sqsQueueUrl              string
	kubernetesDeploymentName string
	kubernetesNamespace      string
)

func Run(p *scale.PodAutoScaler, sqs *sqs.SqsClient, scaleByRatio bool, queuePerPodRatio int) {
	lastScaleUpTime := time.Now()
	lastScaleDownTime := time.Now()
	lastRescaleTime := time.Now()

	for {
		select {
		case <-time.After(pollInterval):
			{
				numMessages, err := sqs.NumMessages()
				if err != nil {
					log.Errorf("Failed to get SQS messages: %v", err)
					continue
				}

				if scaleByRatio {
					deployment, err := p.Client.Deployments(p.Namespace).Get(p.Deployment)
					if err != nil {
						log.Error("Failed to get deployment from kube server, no re-scale occured")
						continue
					}
					currentReplicas := int(deployment.Spec.Replicas)
					d := math.Max(float64(numMessages),1) / float64(queuePerPodRatio)
					newReplicas := int(math.Ceil(d))
					log.WithFields(log.Fields{
						"numMessage": numMessages,
						"d": d,
						"currentReplicas": currentReplicas,
						"newReplicas": newReplicas,
					}).Info("Calculating required pods")

					scaleType := newReplicas - currentReplicas
					if scaleType > 0 { // scale up
						if lastRescaleTime.Add(scaleUpCoolPeriod).After(time.Now()) {
							log.Info("Waiting for cool down, skipping scale up ")
							continue
						}
						log.Info("Scaling up")
						lastRescaleTime = time.Now()
					} else if scaleType < 0 { // scale down
						if lastRescaleTime.Add(scaleDownCoolPeriod).After(time.Now()) {
							log.Info("Waiting for cool down, skipping scale down ")
							continue
						}
						log.Info("Scaling down")
						lastRescaleTime = time.Now()
					} else { // no scale
						log.Info("No scaling required")
						continue
					}

					if err := p.ReScaleByRatio(newReplicas); err != nil {
						log.Errorf("Failed re-scaling: %v", err)
						continue
					}

				} else {
					if numMessages >= scaleUpMessages {
						if lastScaleUpTime.Add(scaleUpCoolPeriod).After(time.Now()) {
							log.Info("Waiting for cool down, skipping scale up ")
							continue
						}

						if err := p.ScaleUp(); err != nil {
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

						if err := p.ScaleDown(); err != nil {
							log.Errorf("Failed scaling down: %v", err)
							continue
						}

						lastScaleDownTime = time.Now()
					}
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
	flag.IntVar(&scaleDownMessages, "scale-down-messages", 10, "Number of messages required to scale down")
	flag.IntVar(&maxPods, "max-pods", 5, "Max pods that kube-sqs-autoscaler can scale")
	flag.IntVar(&minPods, "min-pods", 1, "Min pods that kube-sqs-autoscaler can scale")
	flag.BoolVar(&scaleByRatio, "scale-by-ratio", false, "Use scale by ratio or not")
	flag.IntVar(&queuePerPodRatio, "queue-per-pod-ratio", 100, "Queue per pod ratio")
	flag.StringVar(&awsRegion, "aws-region", "", "Your AWS region")

	flag.StringVar(&sqsQueueUrl, "sqs-queue-url", "", "The sqs queue url")
	flag.StringVar(&kubernetesDeploymentName, "kubernetes-deployment", "", "Kubernetes Deployment to scale. This field is required")
	flag.StringVar(&kubernetesNamespace, "kubernetes-namespace", "default", "The namespace your deployment is running in")

	flag.Parse()

	p := scale.NewPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	sqs := sqs.NewSqsClient(sqsQueueUrl, awsRegion)

	log.Info("Starting kube-sqs-autoscaler")
	Run(p, sqs, scaleByRatio, queuePerPodRatio)
}