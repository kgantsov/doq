package cluster

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Cluster struct {
	namespace        string
	serviceName      string
	serviceDiscovery ServiceDiscovery
	hostname         string
	ip               string
	nodeID           string
	hosts            []string
	httpAddr         string
	raftAddr         string

	clientset           kubernetes.Interface
	inClusterConfigFunc func() (*rest.Config, error)
}

func NewCluster(serviceDiscovery ServiceDiscovery, namespace, ServiceName, httpAddr string) *Cluster {
	c := &Cluster{
		namespace:           namespace,
		serviceName:         ServiceName,
		httpAddr:            httpAddr,
		serviceDiscovery:    serviceDiscovery,
		inClusterConfigFunc: rest.InClusterConfig,
	}

	return c
}

func (c *Cluster) Init() error {
	var err error
	c.hostname, err = c.serviceDiscovery.Hostname()
	if err != nil {
		log.Warn().Msgf("Error getting hostname: %s", err)
		return err
	}

	c.ip, err = c.serviceDiscovery.IP()
	if err != nil {
		log.Error().Msgf("Couldn't lookup the IP: %v\n", err)
		return err
	}

	addrs, err := c.serviceDiscovery.Lookup()
	if err != nil {
		log.Warn().Msgf("Error: %s", err)
		return err
	}

	for _, addr := range addrs {
		log.Debug().Msgf("Discovered address: %s Current host: %s", addr, c.hostname)

		if strings.HasPrefix(addr, c.hostname) {
			c.nodeID = addr
		} else {
			c.hosts = append(c.hosts, addr)
		}
	}

	host, _, err := net.SplitHostPort(c.nodeID)
	if err != nil {
		log.Warn().Msgf("Error splitting host and port: %s %v\n", c.nodeID, err)
	}
	c.raftAddr = fmt.Sprintf("%s:12000", host)

	c.InitKubeClient()

	log.Debug().Msgf(
		"Current node is %s discovered hosts %+v raftAddr %s",
		c.nodeID,
		c.hosts,
		c.raftAddr,
	)

	return nil
}

func (c *Cluster) InitKubeClient() error {
	config, err := c.inClusterConfigFunc()
	if err != nil {
		return err
	}

	c.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) NodeID() string {
	return c.nodeID
}

func (c *Cluster) RaftAddr() string {
	return c.raftAddr
}

func (c *Cluster) Hosts() []string {
	return c.hosts
}

func (c *Cluster) LeaderChanged(isLeader bool) {
	if isLeader {
		ip, err := c.serviceDiscovery.IP()
		if err != nil {
			log.Error().Msgf("Couldn't lookup the IP: %v\n", err)
		}
		c.ip = ip

		log.Info().Msgf("Leader changed, updating EndpointSlice with IP: %s\n", c.ip)

		err = c.UpdateServiceEndpointSlice()
		if err != nil {
			log.Error().Msgf("Failed to update service edpoint sclice: %s", err.Error())
		}
	}
}

func (c *Cluster) UpdateServiceEndpointSlice() error {
	err := c.clientset.DiscoveryV1().EndpointSlices(c.namespace).Delete(
		context.TODO(), c.serviceName, metav1.DeleteOptions{},
	)
	if err != nil {
		return err
	}

	name := "http"
	ready := true
	var port int32 = 80

	i, err := strconv.ParseInt(c.httpAddr, 10, 32)
	if err != nil {
		return err
	}
	port = int32(i)

	newEndpointSlice := &discoveryv1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "discovery.k8s.io/v1",
			Kind:       "EndpointSlice",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.serviceName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"kubernetes.io/service-name": c.serviceName,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{c.ip},
				Conditions: discoveryv1.EndpointConditions{
					Ready: &ready,
				},
				Hostname: &c.hostname,
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name: &name,
				Port: &port,
			},
		},
	}

	createdEndpointSlice, err := c.clientset.DiscoveryV1().EndpointSlices(c.namespace).Create(
		context.TODO(), newEndpointSlice, metav1.CreateOptions{},
	)
	if err != nil {
		return err
	}

	log.Info().Msgf("EndpointSlice %s created successfully!\n", createdEndpointSlice.Name)

	return nil
}
