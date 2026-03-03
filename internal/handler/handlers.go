package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/olegsys/go-shortener/internal/config"
	"github.com/olegsys/go-shortener/internal/model"
)

type Handler struct {
	store *model.URLStore
	cfg   *config.Config
}

func NewHandler(store *model.URLStore, cfg *config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	longURL := string(body)
	id := GenerateID()
	h.store.Urls[id] = longURL
	shortURL := fmt.Sprintf("%s%s", h.cfg.BaseUrl, id)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		fmt.Println("Empty ID")
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}
	longURL, exist := h.store.Urls[id]
	if !exist {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
