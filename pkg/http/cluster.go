package http

import "context"

func (h *Handler) Join(ctx context.Context, input *JoinInput) (*JoinOutput, error) {

	if err := h.node.Join(input.Body.ID, input.Body.Addr); err != nil {
		return &JoinOutput{}, err
	}

	res := &JoinOutput{}
	res.Body.ID = input.Body.ID
	res.Body.Addr = input.Body.Addr

	return res, nil
}
