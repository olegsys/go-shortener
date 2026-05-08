package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/model"
)

type Shortener interface {
	Shorten(ctx context.Context, userID, longURL string) (string, bool, error)
	ShortenBatch(ctx context.Context, userID string, items []model.ShortenBatchItem) ([]model.ShortenBatchResult, error)
	Resolve(ctx context.Context, shortURL string) (string, bool, bool, error)
	GetUserURLs(ctx context.Context, userID string) ([]model.URLPair, error)
	DeleteURLs(ctx context.Context, userID string, ids []string) error
}
type request struct {
	URL string `json:"url"`
}
type resp struct {
	Result string `json:"result"`
}
type deleteTask struct {
	userID  string
	shortID string
}
type Handler struct {
	shortener  Shortener
	deleteChan chan deleteTask
}

func NewHandler(shortener Shortener) *Handler {
	h := &Handler{
		shortener:  shortener,
		deleteChan: make(chan deleteTask, 100),
	}
	go h.runDeleteWorker()
	return h
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sourceURL := string(body)
	if sourceURL == "" {
		http.Error(w, "empty url", http.StatusBadRequest)
		return
	}
	userID := extractUserID(r)
	shortURL, conflict, err := h.shortener.Shorten(r.Context(), userID, sourceURL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if conflict {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if _, err := w.Write([]byte(shortURL)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ShortenJson(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request json", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "empty url", http.StatusBadRequest)
		return
	}
	userID := extractUserID(r)
	shortURL, conflict, err := h.shortener.Shorten(r.Context(), userID, req.URL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	res := resp{Result: shortURL}
	w.Header().Set("Content-Type", "application/json")
	if conflict {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ShortenBatchJson(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var req []model.ShortenBatchItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request json", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}
	for _, item := range req {
		if item.CorrelationID == "" || item.OriginalURL == "" {
			http.Error(w, "invalid batch item", http.StatusBadRequest)
			return
		}
	}

	userID := extractUserID(r)
	batchResult, err := h.shortener.ShortenBatch(r.Context(), userID, req)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(batchResult); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	longURL, exists, isDeleted, err := h.shortener.Resolve(r.Context(), shortURL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "not found", http.StatusBadRequest)
		return
	}
	if isDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}
	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := extractUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.shortener.GetUserURLs(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(urls); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID := extractUserID(r)

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "invalid request json", http.StatusBadRequest)
		return
	}

	for _, id := range ids {
		h.deleteChan <- deleteTask{
			userID:  userID,
			shortID: id,
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) runDeleteWorker() {
	ticker := time.NewTicker(1 * time.Second)
	buffer := make([]deleteTask, 0, 50)
	for {
		select {
		case task := <-h.deleteChan:
			buffer = append(buffer, task)
			if len(buffer) >= 50 {
				h.flush(buffer)
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				h.flush(buffer)
				buffer = buffer[:0]
			}
		}
	}
}

func (h *Handler) flush(tasks []deleteTask) {
	groups := make(map[string][]string)

	for _, t := range tasks {
		groups[t.userID] = append(groups[t.userID], t.shortID)
	}

	for userID, ids := range groups {
		go func(uid string, shortIDs []string) {
			_ = h.shortener.DeleteURLs(context.Background(), uid, shortIDs)
		}(userID, ids)
	}
}

func extractUserID(r *http.Request) string {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return ""
	}
	parts := strings.Split(cookie.Value, "|")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}
