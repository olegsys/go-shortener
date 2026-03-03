package service

import (
	"crypto/rand"
	"encoding/base64"
)

type URLStore interface {
	Set(shortURL, longURL string)
	Get(s string) (string, bool)
}

type ShortenerService struct {
	storage URLStore
	baseURL string
}

func NewShortenerService(storage URLStore, baseURL string) *ShortenerService {
	return &ShortenerService{
		storage: storage,
		baseURL: baseURL,
	}
}

func (s *ShortenerService) Shorten(longURL string) string {
	shortURL := GenerateID()
	s.storage.Set(shortURL, longURL)
	return s.baseURL + "/" + shortURL
}

func (s *ShortenerService) Resolve(shortURL string) (string, bool) {
	return s.storage.Get(shortURL)
}

func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
