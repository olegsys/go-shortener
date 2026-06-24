package handler

import (
	"context"
	"net/http"
)

// Pinger описывает интерфейс для проверки доступности базы данных
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingHandler инкапсулирует логику проверки доступности БД
type PingHandler struct {
	pinger Pinger
}

// NewPingHandler создает новый экземпляр PingHandler
func NewPingHandler(pinger Pinger) *PingHandler {
	return &PingHandler{
		pinger: pinger,
	}
}

// Ping обрабатывает GET-запросы на "/ping"
// Возвращает 200 OK, если БД доступна, иначе 500 Internal Server Error
func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.pinger == nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	if err := h.pinger.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
