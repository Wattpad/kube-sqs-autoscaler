package scale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestScaleUp(t *testing.T) {
	ctx := context.Background()
	p := NewMockPodAutoScaler("deploy", "namespace", 5, 1, 3)

	// Scale up replicas until we reach the max (5).
	// Scale up again and assert that we get an error back when trying to scale up replicas pass the max
	err := p.ScaleUp(ctx)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, int32(4), *deployment.Spec.Replicas)
	err = p.ScaleUp(ctx)
	assert.Nil(t, err)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(5), *deployment.Spec.Replicas)

	err = p.ScaleUp(ctx)
	assert.Nil(t, err)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(5), *deployment.Spec.Replicas)
}

func TestScaleDown(t *testing.T) {
	ctx := context.Background()
	p := NewMockPodAutoScaler("deploy", "namespace", 5, 1, 3)

	err := p.ScaleDown(ctx)
	assert.Nil(t, err)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(2), *deployment.Spec.Replicas)
	err = p.ScaleDown(ctx)
	assert.Nil(t, err)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(1), *deployment.Spec.Replicas)

	err = p.ScaleDown(ctx)
	assert.Nil(t, err)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(1), *deployment.Spec.Replicas)
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int, init int) *PodAutoScaler {
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
	return &PodAutoScaler{
		Client:     mock.AppsV1().Deployments(kubernetesNamespace),
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}
