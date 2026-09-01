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

func NewCluster(
	serviceDiscovery ServiceDiscovery,
	namespace, ServiceName,
	raftAddr string,
	httpAddr string,
) *Cluster {
	c := &Cluster{
		namespace:           namespace,
		serviceName:         ServiceName,
		raftAddr:            raftAddr,
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

	peers, self, err := c.lookupPeers()
	if err != nil {
		log.Warn().Msgf("Error: %s", err)
		return err
	}
	c.hosts = peers
	if self != "" {
		c.nodeID = self
	}

	// On a fresh pod start the headless service SRV records may not yet
	// include this pod itself (EndpointSlice propagation lags behind
	// the container starting), so the loop above can fail to find us and leave
	// nodeID empty. The node's own address is fully derivable from its
	// hostname, service name and namespace, so construct it deterministically
	// instead of crashing with an empty (":9000") Raft bind address.
	if c.nodeID == "" {
		c.nodeID = fmt.Sprintf(
			"%s.%s-internal.%s.svc.cluster.local.:%s",
			c.hostname, c.serviceName, c.namespace, c.httpAddr,
		)
		log.Warn().Msgf(
			"Self not found in SRV discovery yet; derived nodeID=%s", c.nodeID,
		)
	}

	_, raftPort, err := net.SplitHostPort(c.raftAddr)
	if err != nil {
		log.Warn().Msgf("Error splitting host and port for raftAddr: %s %v\n", c.raftAddr, err)
		raftPort = "9000" // Default Raft port
	}

	host, _, err := net.SplitHostPort(c.nodeID)
	if err != nil {
		log.Warn().Msgf("Error splitting host and port for nodeID: %s %v\n", c.nodeID, err)
	}
	c.raftAddr = fmt.Sprintf("%s:%s", host, raftPort)

	c.InitKubeClient()

	log.Debug().Msgf(
		"Current node is %s discovered hosts %+v raftAddr %s",
		c.nodeID,
		c.hosts,
		c.raftAddr,
	)

	return nil
}

// lookupPeers performs a fresh SRV discovery and returns the peer HTTP
// addresses (excluding this node, normalised to the HTTP port and
// de-duplicated) along with this node's own discovered address, if present.
// It is safe to call repeatedly — e.g. while retrying a join — so a node can
// pick up peers that only appeared in DNS after it started. In a StatefulSet
// started with podManagementPolicy: Parallel the SRV records converge over the
// first few seconds, so the initial lookup can be empty or partial.
func (c *Cluster) lookupPeers() (peers []string, self string, err error) {
	addrs, err := c.serviceDiscovery.Lookup()
	if err != nil {
		return nil, "", err
	}

	seen := make(map[string]bool)
	for _, addr := range addrs {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			log.Warn().Msgf("Error splitting host and port for discovered addr %s: %v", addr, err)
			continue
		}
		if port == "0" {
			addr = fmt.Sprintf("%s:%s", host, c.httpAddr)
		}

		log.Debug().Msgf("Discovered address: %s Current host: %s", addr, c.hostname)

		// Identify our own SRV record by the host label, matching either a bare
		// hostname ("doq-0") or an FQDN ("doq-0.doq-internal...."). Comparing the
		// full label (rather than a raw prefix) avoids "doq-1" also matching
		// "doq-10". A headless service publishes one SRV record per named port,
		// so this pod may appear multiple times; record it as self and skip.
		if host == c.hostname || strings.HasPrefix(host, c.hostname+".") {
			self = addr
			continue
		}

		// A headless service publishes one SRV record per named port (raft,
		// grpc, http), so the discovered port may not be the HTTP API port that
		// serves the join endpoint. Peers are always joined over HTTP, so
		// normalise every peer to the configured HTTP port and de-duplicate;
		// otherwise joins target the raft/grpc port and fail, fragmenting the
		// cluster into isolated Raft groups.
		peer := fmt.Sprintf("%s:%s", host, c.httpAddr)
		if seen[peer] {
			continue
		}
		seen[peer] = true
		peers = append(peers, peer)
	}

	return peers, self, nil
}

// DiscoverPeers re-runs SRV discovery and returns the current set of peer HTTP
// addresses, excluding this node. The joiner calls this on every retry so a
// node that started before its peers (or before the seed) were registered in
// DNS still finds them instead of looping over a stale startup snapshot.
func (c *Cluster) DiscoverPeers() ([]string, error) {
	peers, _, err := c.lookupPeers()
	return peers, err
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
		log.Warn().Msgf("Error deleting EndpointSlice: %s", err)
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
