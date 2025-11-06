package http

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

func (h *Handler) Backup(ctx context.Context, input *BackupInput) (*huma.StreamResponse, error) {
	if !h.node.IsLeader() {
		return nil, huma.Error400BadRequest("Only leader can create a backup", nil)
	}

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", "application/octet-stream")

			// Start streaming backup
			writer := ctx.BodyWriter()
			maxVersion, err := h.node.Backup(writer, uint64(0))
			if err != nil {
				log.Error().Err(err).Msg("Failed to create a backup")
				return
			}

			log.Info().Msgf("Backup created with version %d", maxVersion)

			// ✅ Include version metadata in `Content-Disposition`
			ctx.SetHeader("X-Last-Version", fmt.Sprintf("%d", maxVersion))
		},
	}, nil
}

func (h *Handler) Restore(ctx context.Context, input *RestoreInput) (*RestoreOutput, error) {

	if !h.node.IsLeader() {
		return nil, huma.Error400BadRequest("Only leader can restore a backup", nil)
	}

	formData := input.RawBody.Data()

	if err := h.node.Restore(formData.File, 10); err != nil {
		return &RestoreOutput{}, err
	}

	res := &RestoreOutput{}
	res.Body.Status = "SUCCEDED"

	return res, nil
}
