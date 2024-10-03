package logger

import "github.com/rs/zerolog/log"

type BadgerLogger struct {
}

func (bl *BadgerLogger) Errorf(msg string, args ...interface{}) {
	log.Error().Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Warningf(msg string, args ...interface{}) {
	log.Warn().Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Infof(msg string, args ...interface{}) {
	log.Info().Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Debugf(msg string, args ...interface{}) {
	log.Debug().Msgf("[badger] "+msg, args...)
}
