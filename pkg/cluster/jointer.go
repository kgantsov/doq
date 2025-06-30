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
}

func NewJoiner(nodeID, raftAddr string, hosts []string) *Joiner {
	log.Debug().Msgf("Creating new joiner: %s %s %v", nodeID, raftAddr, hosts)
	j := &Joiner{
		nodeID:   nodeID,
		raftAddr: raftAddr,
		hosts:    hosts,
	}

	return j
}

func (j *Joiner) Join() error {
	if len(j.hosts) == 0 {
		log.Debug().Msgf("There is no hosts to join: %d", len(j.hosts))
		return nil
	}

	var host string
	var err error

	for i := 0; i < 3; i++ {
		for _, host = range j.hosts {
			log.Debug().Msgf("Trying to join: %s", host)

			if err = j.join(host, j.raftAddr, j.nodeID); err == nil {
				return nil
			}
		}
		time.Sleep(time.Duration(1) * time.Second)
	}

	return fmt.Errorf("failed to join node at %s: %s", host, err.Error())
}

func (j *Joiner) join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		return err
	}

	log.Debug().Msgf("Joining cluster at %s with data: %s", joinAddr, string(b))

	req, err := http.NewRequest(
		"POST", fmt.Sprintf("http://%s/cluster/join", joinAddr), bytes.NewReader(b),
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
