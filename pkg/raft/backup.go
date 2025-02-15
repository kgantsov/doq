package raft

import (
	"errors"
	"io"

	"github.com/rs/zerolog/log"
)

func (n *Node) Backup(w io.Writer) error {
	_, err := n.db.Backup(w, 0)
	if err != nil {
		return errors.New("Failed to backup database")
	}
	return nil
}

func (n *Node) Restore(r io.Reader) error {
	log.Debug().Msgf("Restoring database from a file %+v", r)
	err := n.db.Load(r, 1000000)
	if err != nil {
		return errors.New("Failed to restore database")
	}
	log.Debug().Msg("Database restored")

	return nil
}
