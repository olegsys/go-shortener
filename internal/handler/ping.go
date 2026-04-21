package handler

import (
	"context"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type PingHandler struct {
	pinger Pinger
}

func NewPingHandler(pinger Pinger) *PingHandler {
	return &PingHandler{
		pinger: pinger,
	}
}

func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.pinger.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
