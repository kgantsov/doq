package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()

		resp, err = handler(ctx, req)

		duration := time.Since(start)
		st := status.Convert(err)

		log.Info().
			Dur("duration", duration).
			Msgf("[grpc] %s - %s error=\"%v\"", info.FullMethod, duration, st.Err())

		return resp, err
	}
}

func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		st := status.Convert(err)

		log.Info().
			Dur("duration", duration).
			Msgf("[grpc] %s - %s error=\"%v\"", info.FullMethod, duration, st.Err())

		return err
	}
}
