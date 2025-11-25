package logger

import "github.com/rs/zerolog/log"

type BadgerLogger struct {
}

func (bl *BadgerLogger) Errorf(msg string, args ...interface{}) {
	log.Error().Str("component", "badger").Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Warningf(msg string, args ...interface{}) {
	log.Warn().Str("component", "badger").Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Infof(msg string, args ...interface{}) {
	log.Info().Str("component", "badger").Msgf("[badger] "+msg, args...)
}

func (bl *BadgerLogger) Debugf(msg string, args ...interface{}) {
	log.Debug().Str("component", "badger").Msgf("[badger] "+msg, args...)
}
