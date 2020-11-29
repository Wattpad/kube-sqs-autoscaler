package scale

import (
	"context"
	"os"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfigPath string
)

type PodAutoScaler struct {
	Client     typedappv1.DeploymentInterface
	Max        int
	Min        int
	Deployment string
	Namespace  string
}

func NewPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *PodAutoScaler {
	kubeConfigPath = os.Getenv("KUBE_CONFIG_PATH")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic("Failed to configure incluster or local config")
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
		log.Infof("More than max pods running. No scale up. Replicas: %d", *deployment.Spec.Replicas)
		return nil
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
		log.Infof("Less than min pods running. No scale down. Replicas: %d", *deployment.Spec.Replicas)
		return nil
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
