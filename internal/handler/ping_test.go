package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockPinger struct {
	pingErr error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestPingHandler_Success(t *testing.T) {
	pinger := &mockPinger{pingErr: nil}
	handler := NewPingHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.Ping(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPingHandler_Failure(t *testing.T) {
	pinger := &mockPinger{pingErr: errors.New("connection failed")}
	handler := NewPingHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.Ping(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database unavailable")
}
