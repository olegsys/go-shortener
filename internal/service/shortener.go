package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/olegsys/go-shortener/internal/model"
)

type URLStore interface {
	Set(ctx context.Context, shortURL, longURL string) (string, bool, error)
	SetBatch(ctx context.Context, pairs []model.URLPair) error
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

func (s *ShortenerService) Shorten(ctx context.Context, longURL string) (string, bool, error) {
	shortURL, err := generateID()
	if err != nil {
		return "", false, fmt.Errorf("generate short id: %w", err)
	}

	storedShortURL, created, err := s.storage.Set(ctx, shortURL, longURL)
	if err != nil {
		return "", false, fmt.Errorf("save short url: %w", err)
	}

	return strings.TrimSuffix(s.baseURL, "/") + "/" + storedShortURL, !created, nil
}

func (s *ShortenerService) ShortenBatch(ctx context.Context, items []model.ShortenBatchItem) ([]model.ShortenBatchResult, error) {
	pairs := make([]model.URLPair, 0, len(items))
	results := make([]model.ShortenBatchResult, 0, len(items))

	for _, item := range items {
		shortID, err := generateID()
		if err != nil {
			return nil, fmt.Errorf("generate batch short id: %w", err)
		}
		pairs = append(pairs, model.URLPair{ShortURL: shortID, LongURL: item.OriginalURL})
		results = append(results, model.ShortenBatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      strings.TrimSuffix(s.baseURL, "/") + "/" + shortID,
		})
	}

	if err := s.storage.SetBatch(ctx, pairs); err != nil {
		return nil, fmt.Errorf("save batch short urls: %w", err)
	}

	return results, nil
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
