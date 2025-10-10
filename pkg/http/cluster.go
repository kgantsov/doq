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

func (h *Handler) Leave(ctx context.Context, input *LeaveInput) (*LeaveOutput, error) {
	if err := h.node.Leave(input.Body.ID); err != nil {
		return &LeaveOutput{}, err
	}

	res := &LeaveOutput{}
	res.Body.ID = input.Body.ID

	return res, nil
}

func (h *Handler) Servers(ctx context.Context, input *ServersInput) (*ServersOutput, error) {
	servers, err := h.node.GetServers()
	if err != nil {
		return &ServersOutput{}, err
	}

	responsServers := make([]Server, len(servers))
	for i, server := range servers {
		responsServers[i] = Server{
			Id:       server.Id,
			Addr:     server.Addr,
			IsLeader: server.IsLeader,
			Suffrage: server.Suffrage,
		}
	}

	res := &ServersOutput{
		Body: ServersOutputBody{
			Servers: responsServers,
		},
	}

	return res, nil
}
