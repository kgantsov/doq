package cluster

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func mockInClusterConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

func mockInClusterConfigError() (*rest.Config, error) {
	return nil, errors.New("in-cluster config error")
}

// TestCluster_Init tests the Init method of the Cluster
func TestCluster_Init(t *testing.T) {
	serviceDiscovery := createMockServiceDiscoverySRV()

	c := NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err := c.Init()

	assert.NoError(t, err)
	assert.Equal(t, "doq-2", c.hostname)
	assert.Equal(t, "51.11.1.2", c.ip)
	assert.Equal(t, []string{"doq-0:8000", "doq-1:8000"}, c.hosts)
}

// TestCluster_InitError tests the Init method of the Cluster with an error
func TestCluster_InitError(t *testing.T) {
	SRVError := errors.New("Error getting SRV records")
	IPError := errors.New("Error getting IP")
	HostnameError := errors.New("Error getting Hostname")

	serviceDiscovery := createMockServiceDiscoverySRV()
	serviceDiscovery.lookupSRVFn = func(service, proto, name string) (string, []*net.SRV, error) {
		return "", nil, SRVError
	}

	c := NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err := c.Init()

	assert.ErrorIs(t, SRVError, err)

	serviceDiscovery = createMockServiceDiscoverySRV()
	serviceDiscovery.lookupIPFn = func(host string) ([]string, error) {
		return nil, IPError
	}

	c = NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err = c.Init()

	assert.ErrorIs(t, IPError, err)

	serviceDiscovery = createMockServiceDiscoverySRV()
	serviceDiscovery.lookupHostnameFn = func() (string, error) {
		return "", HostnameError
	}

	c = NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err = c.Init()

	assert.ErrorIs(t, HostnameError, err)
}

// TestCluster_NodeID tests the NodeID method of the Cluster
func TestCluster_NodeID(t *testing.T) {
	serviceDiscovery := createMockServiceDiscoverySRV()
	serviceDiscovery.lookupHostnameFn = func() (string, error) {
		return "doq-0", nil
	}

	c := NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err := c.Init()
	assert.NoError(t, err)

	assert.Equal(t, "doq-0:8000", c.NodeID())

	serviceDiscovery.lookupHostnameFn = func() (string, error) {
		return "doq-2", nil
	}

	c = NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err = c.Init()
	assert.NoError(t, err)

	assert.Equal(t, "doq-2:8000", c.NodeID())
}

// TestCluster_RaftAddr tests the RaftAddr method of the Cluster
func TestCluster_RaftAddr(t *testing.T) {
	serviceDiscovery := createMockServiceDiscoverySRV()
	serviceDiscovery.lookupHostnameFn = func() (string, error) {
		return "doq-0", nil
	}

	c := NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err := c.Init()
	assert.NoError(t, err)

	assert.Equal(t, "doq-0:12000", c.RaftAddr())

	serviceDiscovery.lookupHostnameFn = func() (string, error) {
		return "doq-2", nil
	}

	c = NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err = c.Init()
	assert.NoError(t, err)

	assert.Equal(t, "doq-2:12000", c.RaftAddr())
}

// TestCluster_Hosts tests the Hosts method of the Cluster
func TestCluster_Hosts(t *testing.T) {
	serviceDiscovery := createMockServiceDiscoverySRV()

	c := NewCluster(
		serviceDiscovery, "test-namespace", "doq", "8000",
	)
	c.inClusterConfigFunc = mockInClusterConfig

	err := c.Init()
	assert.NoError(t, err)

	assert.Equal(t, []string{"doq-0:8000", "doq-1:8000"}, c.Hosts())
}

func createMockServiceDiscoverySRV() *ServiceDiscoverySRV {
	return &ServiceDiscoverySRV{
		namespace:   "test-namespace",
		serviceName: "doq",
		lookupSRVFn: func(service, proto, name string) (string, []*net.SRV, error) {
			return "", []*net.SRV{
				{Target: "doq-0", Port: 8000},
				{Target: "doq-1", Port: 8000},
				{Target: "doq-2", Port: 8000},
			}, nil
		},
		lookupIPFn: func(host string) ([]string, error) {
			return []string{"51.11.1.2"}, nil
		},
		lookupHostnameFn: func() (string, error) {
			return "doq-2", nil
		},
	}
}

// TestUpdateServiceEndpointSlice tests the UpdateServiceEndpointSlice method of the Cluster
func TestInitKubeClient(t *testing.T) {
	tests := []struct {
		name            string
		cluster         Cluster
		inClusterConfig func() (*rest.Config, error)
		setupClient     func() *fake.Clientset
		expectedErr     string
	}{
		{
			name: "successful execution",
			cluster: Cluster{
				namespace:   "default",
				serviceName: "test-service",
				httpAddr:    "8080",
				ip:          "192.168.1.1",
				hostname:    "test-host",
			},
			inClusterConfig: mockInClusterConfig,
			expectedErr:     "",
		},
		{
			name: "error in InClusterConfig",
			cluster: Cluster{
				namespace:   "default",
				serviceName: "test-service",
				httpAddr:    "8080",
				ip:          "192.168.1.1",
				hostname:    "test-host",
			},
			inClusterConfig: mockInClusterConfigError,
			expectedErr:     "in-cluster config error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cluster.inClusterConfigFunc = tt.inClusterConfig

			err := tt.cluster.InitKubeClient()

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestUpdateServiceEndpointSlice(t *testing.T) {
	tests := []struct {
		name            string
		cluster         Cluster
		inClusterConfig func() (*rest.Config, error)
		setupClient     func() *fake.Clientset
		expectedErr     string
	}{
		{
			name: "successful execution",
			cluster: Cluster{
				namespace:   "default",
				serviceName: "test-service",
				httpAddr:    "8080",
				ip:          "192.168.1.1",
				hostname:    "test-host",
			},
			inClusterConfig: mockInClusterConfig,
			setupClient: func() *fake.Clientset {
				client := fake.NewSimpleClientset()
				client.DiscoveryV1().EndpointSlices("default").Create(context.TODO(), &discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
					},
				}, metav1.CreateOptions{})
				return client
			},
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cluster.inClusterConfigFunc = tt.inClusterConfig
			tt.cluster.clientset = tt.setupClient()

			err := tt.cluster.UpdateServiceEndpointSlice()

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestLeaderChanged(t *testing.T) {

	cluster := Cluster{
		namespace:        "default",
		serviceName:      "test-service",
		httpAddr:         "8080",
		ip:               "192.168.1.1",
		hostname:         "test-host",
		serviceDiscovery: createMockServiceDiscoverySRV(),
	}
	inClusterConfig := mockInClusterConfig

	client := fake.NewSimpleClientset()
	client.DiscoveryV1().EndpointSlices("default").Create(context.TODO(), &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})

	cluster.clientset = client
	cluster.inClusterConfigFunc = inClusterConfig

	cluster.LeaderChanged(true)

	actions := client.Actions()

	assert.Len(t, actions, 3)
	assert.Equal(t, "endpointslices", actions[0].GetResource().Resource)
	assert.Equal(t, "create", actions[0].GetVerb())
	assert.Equal(t, cluster.namespace, actions[0].GetNamespace())

	assert.Equal(t, "endpointslices", actions[1].GetResource().Resource)
	assert.Equal(t, "delete", actions[1].GetVerb())
	assert.Equal(t, cluster.namespace, actions[1].GetNamespace())

	assert.Equal(t, "endpointslices", actions[2].GetResource().Resource)
	assert.Equal(t, "create", actions[2].GetVerb())
	assert.Equal(t, cluster.namespace, actions[2].GetNamespace())
}
