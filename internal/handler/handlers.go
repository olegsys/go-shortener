package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Shortener interface {
	Shorten(longURL string) string
	Resolve(shortURL string) (string, bool)
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusBadRequest)
		return
	}
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
