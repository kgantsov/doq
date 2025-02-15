package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) Backup(ctx context.Context, input *BackupInput) (*huma.StreamResponse, error) {
	if !h.node.IsLeader() {
		return nil, huma.Error400BadRequest("Only leader can create a backup", nil)
	}

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			// Write header info before streaming the body.
			ctx.SetHeader("Content-Type", "application/octet-stream")
			writer := ctx.BodyWriter()

			h.node.Backup(writer)
		},
	}, nil
}

func (h *Handler) Restore(ctx context.Context, input *RestoreInput) (*RestoreOutput, error) {

	if !h.node.IsLeader() {
		return nil, huma.Error400BadRequest("Only leader can restore a backup", nil)
	}

	formData := input.RawBody.Data()

	if err := h.node.Restore(formData.File); err != nil {
		return &RestoreOutput{}, err
	}

	res := &RestoreOutput{}
	res.Body.Status = "SUCCEDED"

	return res, nil
}
