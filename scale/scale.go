package scale

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type PodAutoScaler struct {
	Client     typedappv1.DeploymentInterface
	Max        int
	Min        int
	Deployment string
	Namespace  string
}

func NewPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *PodAutoScaler {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		panic("Failed to configure incluster config")
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("Failed to configure client")
	}

	return &PodAutoScaler{
		Client:     k8sClient.AppsV1().Deployments(kubernetesNamespace),
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}

func (p *PodAutoScaler) ScaleUp(ctx context.Context) error {
	deployment, err := p.Client.Get(ctx, p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale up occured")
	}

	currentReplicas := deployment.Spec.Replicas

	if *currentReplicas >= int32(p.Max) {
		return errors.New("Max pods reached")
	}
	nextReplicas := *currentReplicas + int32(1)
	deployment.Spec.Replicas = &nextReplicas

	_, err = p.Client.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale up")
	}

	log.Infof("Scale up successful. Replicas: %d", *deployment.Spec.Replicas)
	return nil
}

func (p *PodAutoScaler) ScaleDown(ctx context.Context) error {
	deployment, err := p.Client.Get(ctx, p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale down occured")
	}

	currentReplicas := deployment.Spec.Replicas

	if *currentReplicas <= int32(p.Min) {
		return errors.New("Min pods reached")
	}

	nextReplicas := *currentReplicas - int32(1)
	deployment.Spec.Replicas = &nextReplicas

	_, err = p.Client.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale down")
	}

	log.Infof("Scale down successful. Replicas: %d", *deployment.Spec.Replicas)
	return nil
}
