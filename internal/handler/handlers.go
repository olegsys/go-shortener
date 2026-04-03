package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Shortener interface {
	Shorten(longURL string) string
	Resolve(shortURL string) (string, bool)
}
type request struct {
	URL string `json:"url"`
}
type resp struct {
	Result string `json:"result"`
}
type Handler struct {
	shortener Shortener
}

func NewHandler(shortener Shortener) *Handler {
	return &Handler{
		shortener: shortener,
	}
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
	fmt.Println("Input URL:", sourceURL)

	shortURL := h.shortener.Shorten(sourceURL)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
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
	}
	shortURL := h.shortener.Shorten(req.URL)
	res := resp{Result: shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	longURL, exists := h.shortener.Resolve(shortURL)
	if !exists {
		http.Error(w, "not found", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
