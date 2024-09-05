package cluster

import (
	"fmt"
	"net"
	"os"
)

type ServiceDiscovery interface {
	Lookup() ([]string, error)
	IP() (string, error)
	Hostname() (string, error)
}

type ServiceDiscoverySRV struct {
	namespace   string
	serviceName string

	// Overrides for testing purposes
	lookupSRVFn      func(service, proto, name string) (string, []*net.SRV, error)
	lookupIPFn       func(host string) ([]string, error)
	lookupHostnameFn func() (string, error)
}

func NewServiceDiscoverySRV(namespace, serviceName string) *ServiceDiscoverySRV {
	return &ServiceDiscoverySRV{
		namespace:   namespace,
		serviceName: serviceName,

		lookupSRVFn:      net.LookupSRV,
		lookupIPFn:       net.LookupHost,
		lookupHostnameFn: os.Hostname,
	}
}

func (s *ServiceDiscoverySRV) Lookup() ([]string, error) {
	_, addrs, err := s.lookupSRVFn(
		"", "", fmt.Sprintf("%s-internal.%s.svc.cluster.local", s.serviceName, s.namespace),
	)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for _, srv := range addrs {
		hosts = append(hosts, fmt.Sprintf("%s:%d", srv.Target, srv.Port))
	}

	return hosts, nil
}

func (s *ServiceDiscoverySRV) IP() (string, error) {
	hostname, err := s.Hostname()
	if err != nil {
		return "", err
	}

	addrs, err := s.lookupIPFn(hostname)
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		return a, nil
	}
	return "", fmt.Errorf("Couldn't locate the IP for the service: %s", s.serviceName)

}

func (s *ServiceDiscoverySRV) Hostname() (string, error) {
	var err error
	hostname, err := s.lookupHostnameFn()
	if err != nil {
		return "", err
	}

	return hostname, nil
}
