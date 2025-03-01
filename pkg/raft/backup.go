package raft

import (
	"errors"
	"io"

	"github.com/rs/zerolog/log"
)

func (n *Node) Backup(w io.Writer, since uint64) (uint64, error) {
	log.Info().Msgf("Backing up database from version %d", since)
	maxVersion, err := n.db.Backup(w, 0)
	if err != nil {
		return 0, errors.New("Failed to backup database")
	}
	log.Info().Msgf("Database backed up from version %d to %d", since, maxVersion)
	return maxVersion, nil
}

func (n *Node) Restore(r io.Reader, maxPendingWrites int) error {
	log.Info().Msgf("Restoring database from a file wih maxPendingWrites %d", maxPendingWrites)
	err := n.db.Load(r, maxPendingWrites)
	if err != nil {
		return errors.New("Failed to restore database")
	}
	log.Info().Msgf("Database restored successfully with maxPendingWrites %d", maxPendingWrites)
	return nil
}
