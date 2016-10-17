package scale

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

func TestScaleUp(t *testing.T) {
	p := NewMockPodAutoScaler("test", "test", 5, 1)

	// Scale up replicas until we reach the max (5).
	// Scale up again and assert that we get an error back when trying to scale up replicas pass the max
	err := p.ScaleUp()
	deployment, _ := p.Client.Deployments("test").Get("test")
	assert.Nil(t, err)
	assert.Equal(t, int32(4), deployment.Spec.Replicas)
	err = p.ScaleUp()
	assert.Nil(t, err)
	deployment, _ = p.Client.Deployments("test").Get("test")
	assert.Equal(t, int32(5), deployment.Spec.Replicas)

	err = p.ScaleUp()
	assert.NotNil(t, err)
	deployment, _ = p.Client.Deployments("test").Get("test")
	assert.Equal(t, int32(5), deployment.Spec.Replicas)
}

func TestScaleDown(t *testing.T) {
	p := NewMockPodAutoScaler("test", "test", 5, 1)

	err := p.ScaleDown()
	assert.Nil(t, err)
	deployment, _ := p.Client.Deployments("test").Get("test")
	assert.Equal(t, int32(2), deployment.Spec.Replicas)
	err = p.ScaleDown()
	assert.Nil(t, err)
	deployment, _ = p.Client.Deployments("test").Get("test")
	assert.Equal(t, int32(1), deployment.Spec.Replicas)

	err = p.ScaleDown()
	assert.NotNil(t, err)
	deployment, _ = p.Client.Deployments("test").Get("test")
	assert.Equal(t, int32(1), deployment.Spec.Replicas)
}

type MockDeployment struct {
	client *MockKubeClient
}

type MockKubeClient struct {
	// stores the state of Deployment as if the api server did
	Deployment *extensions.Deployment
}

func (m *MockDeployment) Get(name string) (*extensions.Deployment, error) {
	return m.client.Deployment, nil
}

func (m *MockDeployment) Update(deployment *extensions.Deployment) (*extensions.Deployment, error) {
	m.client.Deployment.Spec.Replicas = deployment.Spec.Replicas
	return m.client.Deployment, nil
}

func (m *MockDeployment) List(opts api.ListOptions) (*extensions.DeploymentList, error) {
	return nil, nil
}

func (m *MockDeployment) Delete(name string, options *api.DeleteOptions) error {
	return nil
}

func (m *MockDeployment) Create(*extensions.Deployment) (*extensions.Deployment, error) {
	return nil, nil
}

func (m *MockDeployment) UpdateStatus(*extensions.Deployment) (*extensions.Deployment, error) {
	return nil, nil
}

func (m *MockDeployment) Watch(opts api.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (m *MockDeployment) Rollback(*extensions.DeploymentRollback) error {
	return nil
}

func (m *MockKubeClient) Deployments(namespace string) kclient.DeploymentInterface {
	return &MockDeployment{
		client: m,
	}
}

func NewMockKubeClient() *MockKubeClient {
	return &MockKubeClient{
		Deployment: &extensions.Deployment{
			Spec: extensions.DeploymentSpec{
				Replicas: 3,
			},
		},
	}
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *PodAutoScaler {
	mockClient := NewMockKubeClient()

	return &PodAutoScaler{
		Client:     mockClient,
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}
