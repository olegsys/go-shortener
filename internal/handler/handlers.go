// Package handler реализует HTTP-обработчики запросов
package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/audit"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/model"
)

// Shortener описывает интерфейс сервиса сокращения URL, используемый хендлерами
type Shortener interface {
	Shorten(ctx context.Context, longURL string) (string, bool, error)
	ShortenBatch(ctx context.Context, items []model.ShortenBatchItem) ([]model.ShortenBatchResult, error)
	Resolve(ctx context.Context, shortURL string) (string, bool, bool, error)
	GetUserURLs(ctx context.Context) ([]model.URLPair, error)
	DeleteURLs(ctx context.Context, userID string, ids []string) error
	Stats(ctx context.Context) (int, int, error)
}
type deletionService interface {
	Enqueue(userID, shortID string) error
}

type request struct {
	URL string `json:"url"`
}
type resp struct {
	Result string `json:"result"`
}

// Handler группирует HTTP-хендлеры сервиса сокращения URL
type Handler struct {
	shortener   Shortener
	deletionSvc deletionService
	auditor     audit.Auditor
}

// NewHandler создает новый экземпляр Handler с заданными зависимостями
func NewHandler(shortener Shortener, deletionSvc deletionService, auditor audit.Auditor) *Handler {
	if auditor == nil {
		auditor = &audit.NoopAuditor{}
	}
	return &Handler{
		shortener:   shortener,
		deletionSvc: deletionSvc,
		auditor:     auditor,
	}
}

// Shorten обрабатывает POST-запросы на корневой путь "/"
// Принимает оригинальный URL в теле запроса в формате text/plain
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
	shortURL, conflict, err := h.shortener.Shorten(r.Context(), sourceURL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	h.auditor.Publish(r.Context(), audit.Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    sourceURL,
	})

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

// ShortenJSON обрабатывает POST-запросы на "/api/shorten"
// Принимает JSON вида {"url": "..."}
func (h *Handler) ShortenJSON(w http.ResponseWriter, r *http.Request) {
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
	shortURL, conflict, err := h.shortener.Shorten(r.Context(), req.URL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	h.auditor.Publish(r.Context(), audit.Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    req.URL,
	})

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

// ShortenBatchJSON обрабатывает POST-запросы на "/api/shorten/batch"
// Принимает массив JSON-объектов для пакетного сокращения
func (h *Handler) ShortenBatchJSON(w http.ResponseWriter, r *http.Request) {
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
	batchResult, err := h.shortener.ShortenBatch(r.Context(), req)
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

// Redirect обрабатывает GET-запросы на "/{id}"
// Выполняет HTTP-редирект на оригинальный URL
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

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	h.auditor.Publish(r.Context(), audit.Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    longURL,
	})

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// GetUserURLs обрабатывает GET-запросы на "/api/user/urls"
// Возвращает все URL текущего пользователя
func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.shortener.GetUserURLs(r.Context())
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

// DeleteURLs обрабатывает DELETE-запросы на "/api/user/urls"
// Помечает указанные URL как удаленные
func (h *Handler) DeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "invalid request json", http.StatusBadRequest)
		return
	}

	for _, id := range ids {
		if err := h.deletionSvc.Enqueue(userID, id); err != nil {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// statsResponse описывает ответ эндпоинта статистики
type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// Stats обрабатывает GET-запросы на "/api/internal/stats"
// Возвращает количество сокращённых URL и пользователей в сервисе
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	urls, users, err := h.shortener.Stats(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statsResponse{URLs: urls, Users: users}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
