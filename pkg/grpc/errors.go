package grpc

import (
	"errors"
	"fmt"

	doqerrors "github.com/kgantsov/doq/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapError translates a domain error coming from the raft/queue layers into a
// gRPC status with a code and message that reflect its real cause (e.g.
// "queue not found" -> codes.NotFound) instead of a single generic
// codes.Unknown for everything. The underlying error is preserved via
// errors.Is/Unwrap all the way from where it originated, so it is matched
// here rather than rebuilt.
func mapError(detail string, err error) error {
	switch {
	case errors.Is(err, doqerrors.ErrQueueNotFound),
		errors.Is(err, doqerrors.ErrMessageNotFound),
		errors.Is(err, doqerrors.ErrEmptyQueue):
		return status.Error(codes.NotFound, fmt.Sprintf("%s: %s", detail, err))
	case errors.Is(err, doqerrors.ErrInvalidStrategy),
		errors.Is(err, doqerrors.ErrInvalidAckTimeout),
		errors.Is(err, doqerrors.ErrInvalidMaxUnacked),
		errors.Is(err, doqerrors.ErrInvalidQueueSettings):
		return status.Error(codes.InvalidArgument, fmt.Sprintf("%s: %s", detail, err))
	}

	// A non-leader node proxies writes to the leader over gRPC (see
	// pkg/raft/commands.go), so err may already be a gRPC status error
	// classified by the leader's own mapError call rather than one of the
	// local sentinels above — a Go sentinel's identity doesn't survive gRPC
	// serialization. Preserve that classification instead of collapsing it
	// to Internal.
	if s, ok := status.FromError(err); ok {
		return status.Error(s.Code(), fmt.Sprintf("%s: %s", detail, s.Message()))
	}

	return status.Error(codes.Internal, fmt.Sprintf("%s: %s", detail, err))
}
