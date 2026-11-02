//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// startTimeout bounds how long a node may take to start answering HTTP.
	startTimeout = 15 * time.Second
	// electionTimeout bounds how long the cluster may take to agree on a leader.
	electionTimeout = 20 * time.Second
	// stopTimeout bounds a graceful shutdown before the node is killed.
	stopTimeout = 15 * time.Second
	// logTailBytes is how much of each node's log is printed when a test fails.
	logTailBytes = 8 << 10
)

// Node is one doq server process. Its ports and data directory stay the same
// across restarts, so a restarted node rejoins with its persisted state.
type Node struct {
	ID       string
	HTTPPort int
	RaftAddr string
	GRPCAddr string
	JoinAddr string // empty for the node that bootstraps the cluster
	// ExtraArgs are appended to the node's command line on every start.
	ExtraArgs []string

	dataDir string
	logPath string
	cmd     *exec.Cmd
	exited  chan struct{}
	client  *Client
}

// Cluster is a set of nodes started by StartCluster.
type Cluster struct {
	Nodes []*Node
}

// ClusterOption configures a cluster's nodes before they start.
type ClusterOption func(*Cluster)

// WithNodeArgs adds command-line flags to node i.
func WithNodeArgs(i int, args ...string) ClusterOption {
	return func(c *Cluster) {
		c.Nodes[i].ExtraArgs = append(c.Nodes[i].ExtraArgs, args...)
	}
}

// StartCluster starts an n-node cluster: node-0 bootstraps it and the others
// join through node-0. It returns once every node is a member and they agree
// on a leader. All nodes are killed when the test ends.
func StartCluster(t *testing.T, n int, opts ...ClusterOption) *Cluster {
	t.Helper()

	dataDir := t.TempDir()
	c := &Cluster{}
	for i := 0; i < n; i++ {
		node := &Node{
			ID:       fmt.Sprintf("node-%d", i),
			HTTPPort: freePort(t),
			RaftAddr: fmt.Sprintf("localhost:%d", freePort(t)),
			GRPCAddr: fmt.Sprintf("localhost:%d", freePort(t)),
			dataDir:  dataDir,
			logPath:  filepath.Join(dataDir, fmt.Sprintf("node-%d.log", i)),
		}
		if i > 0 {
			node.JoinAddr = c.Nodes[0].HTTPAddr()
		}
		node.client = NewClient(node.HTTPAddr())
		c.Nodes = append(c.Nodes, node)
	}
	for _, opt := range opts {
		opt(c)
	}

	t.Cleanup(func() {
		for _, node := range c.Nodes {
			node.Kill(t)
		}
		if t.Failed() {
			c.dumpLogs(t)
		}
	})

	// The seed must be leader before the others try to join it.
	c.Nodes[0].Start(t)
	c.WaitForLeader(t)
	for _, node := range c.Nodes[1:] {
		node.Start(t)
	}
	c.WaitForMembers(t, n)
	c.WaitForLeader(t)

	return c
}

// HTTPAddr is the node's HTTP host:port.
func (n *Node) HTTPAddr() string {
	return fmt.Sprintf("localhost:%d", n.HTTPPort)
}

// Client returns an HTTP client bound to this node.
func (n *Node) Client() *Client {
	return n.client
}

// Running reports whether the node's process is alive.
func (n *Node) Running() bool {
	if n.cmd == nil {
		return false
	}
	select {
	case <-n.exited:
		return false
	default:
		return true
	}
}

// Start launches the node's process and waits until it answers HTTP. Output
// is appended to the node's log file, so it survives restarts.
func (n *Node) Start(t *testing.T) {
	t.Helper()
	require.False(t, n.Running(), "%s is already running", n.ID)

	args := []string{
		"--cluster.node_id", n.ID,
		"--http.port", strconv.Itoa(n.HTTPPort),
		"--raft.address", n.RaftAddr,
		"--grpc.address", n.GRPCAddr,
		"--storage.data_dir", n.dataDir,
		"--logging.level", "info",
	}
	if n.JoinAddr != "" {
		args = append(args, "--cluster.join_addr", n.JoinAddr)
	}
	args = append(args, n.ExtraArgs...)

	logFile, err := os.OpenFile(n.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, err)
	fmt.Fprintf(logFile, "\n===== starting %s at %s =====\n", n.ID, time.Now().Format(time.RFC3339Nano))

	cmd := exec.Command(doqBinary, args...)
	// Run outside the repo so the server doesn't pick up a stray config.yaml.
	cmd.Dir = n.dataDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	require.NoError(t, cmd.Start(), "starting %s", n.ID)

	n.cmd = cmd
	n.exited = make(chan struct{})
	go func() {
		_ = cmd.Wait()
		logFile.Close()
		close(n.exited)
	}()

	require.Eventually(t, func() bool {
		if !n.Running() {
			return true // reported below instead of waiting out the timeout
		}
		return n.client.Alive()
	}, startTimeout, 100*time.Millisecond, "%s did not start answering HTTP", n.ID)
	require.True(t, n.Running(), "%s exited during startup, see %s", n.ID, n.logPath)
}

// Kill stops the node with SIGKILL, like a crash: no leadership hand-off and
// no flushing. It is a no-op if the node isn't running.
func (n *Node) Kill(t *testing.T) {
	t.Helper()
	if !n.Running() {
		return
	}
	_ = n.cmd.Process.Kill()
	<-n.exited
}

// Stop shuts the node down gracefully with SIGTERM. Note that a graceful
// shutdown hands leadership to another node first; use Kill to force an
// election.
func (n *Node) Stop(t *testing.T) {
	t.Helper()
	if !n.Running() {
		return
	}
	require.NoError(t, n.cmd.Process.Signal(syscall.SIGTERM))
	select {
	case <-n.exited:
	case <-time.After(stopTimeout):
		n.Kill(t)
		t.Fatalf("%s did not stop within %s and was killed", n.ID, stopTimeout)
	}
}

// LatestSnapshot returns when the node created its most recent finished Raft
// snapshot, or ok=false if it has none. A node that has a snapshot restores
// from it on its next start and replays only the log entries after it.
func (n *Node) LatestSnapshot(t *testing.T) (created time.Time, ok bool) {
	t.Helper()
	// hashicorp/raft's FileSnapshotStore keeps each snapshot in
	// <raft dir>/snapshots/<term>-<index>-<unix ms>/, with a ".tmp" suffix
	// while it's being written.
	dirs, err := filepath.Glob(filepath.Join(n.dataDir, n.ID, "raft", "snapshots", "*", "meta.json"))
	require.NoError(t, err)
	for _, meta := range dirs {
		name := filepath.Base(filepath.Dir(meta))
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		parts := strings.Split(name, "-")
		require.Len(t, parts, 3, "unexpected snapshot directory %s", name)
		ms, err := strconv.ParseInt(parts[2], 10, 64)
		require.NoError(t, err, "unexpected snapshot directory %s", name)
		if at := time.UnixMilli(ms); at.After(created) {
			created, ok = at, true
		}
	}
	return created, ok
}

// Running returns the nodes whose process is alive.
func (c *Cluster) Running() []*Node {
	var nodes []*Node
	for _, n := range c.Nodes {
		if n.Running() {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// Leader returns the current leader if every running node agrees on it, or
// nil if there is no leader yet or the nodes disagree.
func (c *Cluster) Leader() *Node {
	var leader *Node
	for _, n := range c.Running() {
		servers, err := n.client.Servers()
		if err != nil {
			return nil
		}
		var seen *Node
		for _, s := range servers {
			if s.IsLeader {
				seen = c.nodeByRaftAddr(s.Addr)
			}
		}
		if seen == nil || !seen.Running() || (leader != nil && seen != leader) {
			return nil
		}
		leader = seen
	}
	return leader
}

// WaitForLeader waits until every running node agrees on a leader and
// returns it.
func (c *Cluster) WaitForLeader(t *testing.T) *Node {
	t.Helper()
	var leader *Node
	require.Eventually(t, func() bool {
		leader = c.Leader()
		return leader != nil
	}, electionTimeout, 200*time.Millisecond, "running nodes did not agree on a leader")
	return leader
}

// WaitForNewLeader waits for a leader other than old.
func (c *Cluster) WaitForNewLeader(t *testing.T, old *Node) *Node {
	t.Helper()
	var leader *Node
	require.Eventually(t, func() bool {
		leader = c.Leader()
		return leader != nil && leader != old
	}, electionTimeout, 200*time.Millisecond, "no new leader replaced %s", old.ID)
	return leader
}

// WaitForMembers waits until the leader's Raft configuration has n servers.
func (c *Cluster) WaitForMembers(t *testing.T, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		leader := c.Leader()
		if leader == nil {
			return false
		}
		servers, err := leader.client.Servers()
		return err == nil && len(servers) == n
	}, electionTimeout, 200*time.Millisecond, "cluster did not reach %d members", n)
}

// WaitForQueueCounts waits until every running node reports the given counts
// for a queue. Counts are read from each node's own replica, so this also
// waits for followers to catch up.
func (c *Cluster) WaitForQueueCounts(t *testing.T, queue string, want QueueCounts) {
	t.Helper()
	got := map[string]QueueCounts{}
	require.Eventually(t, func() bool {
		ok := true
		for _, n := range c.Running() {
			counts, err := n.client.QueueCounts(queue)
			if err != nil {
				ok = false
				continue
			}
			got[n.ID] = counts
			ok = ok && counts == want
		}
		return ok
	}, 10*time.Second, 100*time.Millisecond, "queue %s counts: want %+v on every node, got %+v", queue, want, got)
}

func (c *Cluster) nodeByRaftAddr(addr string) *Node {
	for _, n := range c.Nodes {
		if n.RaftAddr == addr {
			return n
		}
	}
	return nil
}

func (c *Cluster) dumpLogs(t *testing.T) {
	for _, n := range c.Nodes {
		data, err := os.ReadFile(n.logPath)
		if err != nil {
			continue
		}
		if len(data) > logTailBytes {
			data = append([]byte("...\n"), data[len(data)-logTailBytes:]...)
		}
		t.Logf("----- %s log (%s) -----\n%s", n.ID, n.logPath, bytes.TrimSpace(data))
	}
}

// Ports are picked below the OS's ephemeral range (from 32768 on Linux, 49152
// on macOS). Ports handed out by the OS with ":0" come from that range, which
// is also where outgoing connections get their local port, so a node's
// connection could take a port reserved for a node that hasn't started yet.
const (
	minPort = 20000
	maxPort = 32000
)

var (
	portsMu   sync.Mutex
	usedPorts = map[int]bool{}
)

// freePort returns a port that is free now and that no other node in this
// test run uses. Another process could still take it before the node binds
// it, but in practice that's rare enough for tests.
func freePort(t *testing.T) int {
	t.Helper()
	portsMu.Lock()
	defer portsMu.Unlock()
	for attempt := 0; attempt < 100; attempt++ {
		port := minPort + rand.IntN(maxPort-minPort)
		if usedPorts[port] {
			continue
		}
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		l.Close()
		usedPorts[port] = true
		return port
	}
	t.Fatal("no free port found")
	return 0
}
