package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

type URLStore interface {
	Set(ctx context.Context, shortURL, longURL string) error
	Get(ctx context.Context, s string) (string, bool, error)
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

func (s *ShortenerService) Shorten(ctx context.Context, longURL string) (string, error) {
	shortURL, err := generateID()
	if err != nil {
		return "", fmt.Errorf("generate short id: %w", err)
	}

	if err := s.storage.Set(ctx, shortURL, longURL); err != nil {
		return "", fmt.Errorf("save short url: %w", err)
	}

	return strings.TrimSuffix(s.baseURL, "/") + "/" + shortURL, nil
}

func (s *ShortenerService) Resolve(ctx context.Context, shortURL string) (string, bool, error) {
	longURL, exists, err := s.storage.Get(ctx, shortURL)
	if err != nil {
		return "", false, fmt.Errorf("resolve short url: %w", err)
	}
	return longURL, exists, nil
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}
