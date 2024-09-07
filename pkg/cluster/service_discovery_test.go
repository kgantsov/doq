package cluster

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServiceDiscoverySRV_Lookup tests the Lookup method of the ServiceDiscoverySRV
func TestServiceDiscoverySRV_Lookup(t *testing.T) {
	s := NewServiceDiscoverySRV("test-namespace", "test-service")
	s.lookupSRVFn = func(service, proto, name string) (string, []*net.SRV, error) {
		return "", []*net.SRV{
			{Target: "test-target", Port: 1234},
		}, nil
	}

	hosts, err := s.Lookup()

	assert.NoError(t, err)
	assert.Len(t, hosts, 1)
	assert.Equal(t, "test-target:1234", hosts[0])
}

// TestServiceDiscoverySRV_IP tests the IP method of the ServiceDiscoverySRV
func TestServiceDiscoverySRV_IP(t *testing.T) {
	s := NewServiceDiscoverySRV("test-namespace", "test-service")
	s.lookupIPFn = func(host string) ([]string, error) {
		return []string{"test-ip"}, nil
	}

	ip, err := s.IP()

	assert.NoError(t, err)
	assert.Equal(t, "test-ip", ip)
}

// TestServiceDiscoverySRV_Hostname tests the Hostname method of the ServiceDiscoverySRV
func TestServiceDiscoverySRV_Hostname(t *testing.T) {
	s := NewServiceDiscoverySRV("test-namespace", "test-service")
	s.lookupHostnameFn = func() (string, error) {
		return "test-hostname", nil
	}

	hostname, err := s.Hostname()

	assert.NoError(t, err)
	assert.Equal(t, "test-hostname", hostname)
}
