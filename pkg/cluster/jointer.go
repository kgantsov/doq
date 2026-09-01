package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Joiner struct {
	nodeID   string
	raftAddr string
	hosts    []string

	// resolve re-discovers peers on each retry round. It may be nil (e.g. a
	// non-Kubernetes node joining a static address), in which case the joiner
	// falls back to the hosts captured at construction.
	resolve func() ([]string, error)

	maxAttempts   int
	retryInterval time.Duration
}

func NewJoiner(nodeID, raftAddr string, hosts []string, resolve func() ([]string, error)) *Joiner {
	log.Debug().Msgf("Creating new joiner: %s %s %v", nodeID, raftAddr, hosts)
	j := &Joiner{
		nodeID:   nodeID,
		raftAddr: raftAddr,
		hosts:    hosts,
		resolve:  resolve,
		// ~5 minutes of retries at 2s. On a Parallel StatefulSet start the seed
		// may not have won its election (or even appeared in DNS) when the other
		// pods first try to join, so the window must comfortably outlast DNS
		// convergence and leader election.
		maxAttempts:   150,
		retryInterval: 2 * time.Second,
	}

	return j
}

// Join admits this node into an existing cluster. It re-resolves the peer set
// on every round (when a resolver is configured) rather than looping over a
// stale startup snapshot: a node that started before its peers — or before the
// ordinal-0 seed became leader — must still find them. An empty peer set is
// treated as "not discovered yet" and retried, not as success; otherwise a
// fresh pod whose SRV records have not converged would silently give up and sit
// isolated forever.
func (j *Joiner) Join() error {
	// No peers and no way to discover any (non-Kubernetes single node): there is
	// genuinely nothing to join, so preserve the historical no-op behavior.
	if j.resolve == nil && len(j.hosts) == 0 {
		log.Debug().Msg("There is no hosts to join and no resolver; nothing to do")
		return nil
	}

	var lastErr error

	for i := 0; i < j.maxAttempts; i++ {
		hosts := j.hosts
		if j.resolve != nil {
			if fresh, err := j.resolve(); err != nil {
				log.Warn().Msgf("Error re-resolving peers to join: %s", err)
			} else if len(fresh) > 0 {
				hosts = fresh
			}
		}

		if len(hosts) == 0 {
			lastErr = fmt.Errorf("no peers discovered yet")
			log.Debug().Msg("No peers discovered yet; waiting before retrying join")
			time.Sleep(j.retryInterval)
			continue
		}

		for _, host := range hosts {
			log.Debug().Msgf("Trying to join: %s", host)

			if err := j.join(host, j.raftAddr, j.nodeID); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		time.Sleep(j.retryInterval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no peers discovered")
	}
	return fmt.Errorf("failed to join cluster after %d attempts: %w", j.maxAttempts, lastErr)
}

func (j *Joiner) join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		return err
	}

	log.Debug().Msgf("Joining cluster at %s with data: %s", joinAddr, string(b))

	req, err := http.NewRequest(
		"POST", fmt.Sprintf("http://%s/API/v1/cluster/join", joinAddr), bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to join: %s", joinAddr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response body: %w", err)
	}
	log.Info().Msgf("JOINED %+v %+v", resp.StatusCode, string(body))
	return nil
}
