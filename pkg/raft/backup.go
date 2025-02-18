package raft

import (
	"errors"
	"io"

	"github.com/rs/zerolog/log"
)

func (n *Node) Backup(w io.Writer, since uint64) (uint64, error) {
	maxVersion, err := n.db.Backup(w, 0)
	if err != nil {
		return 0, errors.New("Failed to backup database")
	}
	return maxVersion, nil
}

func (n *Node) Restore(r io.Reader, maxPendingWrites int) error {
	log.Debug().Msgf("Restoring database from a file %+v", r)
	err := n.db.Load(r, maxPendingWrites)
	if err != nil {
		return errors.New("Failed to restore database")
	}
	log.Debug().Msg("Database restored")

	return nil
}
