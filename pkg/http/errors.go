package http

import (
	"errors"

	"github.com/danielgtaylor/huma/v2"
	doqerrors "github.com/kgantsov/doq/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapError translates a domain error coming from the raft/queue layers into a
// huma error with a status code and message that reflect its real cause
// (e.g. "queue not found" -> 404) instead of a single generic 400 for
// everything. The underlying error is preserved via errors.Is/Unwrap all the
// way from where it originated, so it is matched here rather than rebuilt.
func mapError(detail string, err error) error {
	switch {
	case errors.Is(err, doqerrors.ErrQueueNotFound),
		errors.Is(err, doqerrors.ErrMessageNotFound),
		errors.Is(err, doqerrors.ErrEmptyQueue):
		return huma.Error404NotFound(detail, err)
	case errors.Is(err, doqerrors.ErrInvalidStrategy),
		errors.Is(err, doqerrors.ErrInvalidAckTimeout),
		errors.Is(err, doqerrors.ErrInvalidMaxUnacked),
		errors.Is(err, doqerrors.ErrInvalidQueueSettings):
		return huma.Error422UnprocessableEntity(detail, err)
	case errors.Is(err, doqerrors.ErrNoRaftLeader):
		return huma.Error503ServiceUnavailable(detail, err)
	}

	// A non-leader node proxies writes to the leader over gRPC (see
	// pkg/raft/commands.go), so err may already be a gRPC status error
	// classified by the leader's pkg/grpc/errors.go mapError rather than one
	// of the local sentinels above — a Go sentinel's identity doesn't survive
	// gRPC serialization. Preserve that classification instead of collapsing
	// it to a generic 500.
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.NotFound:
			return huma.Error404NotFound(detail, err)
		case codes.InvalidArgument:
			return huma.Error422UnprocessableEntity(detail, err)
		}
	}

	return huma.Error500InternalServerError(detail, err)
}
