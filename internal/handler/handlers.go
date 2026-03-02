package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/olegsys/go-shortener/internal/model"
)

type Handler struct {
	Store *model.URLStore
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
	longUrl := string(body)
	id := GenerateID()
	h.Store.Urls[id] = longUrl
	shortURL := fmt.Sprintf("http://localhost:8080/%s", id)
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
	longURL, exist := h.Store.Urls[id]
	if !exist {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
