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
	SetBatch(ctx context.Context, pairs []model.URLPair) ([]model.URLPair, error)
	Get(ctx context.Context, s string) (string, bool, error)
	GetUserURLs(ctx context.Context) ([]model.URLPair, error)
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
	const maxRetries = 10
	var storedShortURL string
	var created bool
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		shortURL, err := generateID()
		if err != nil {
			return "", false, fmt.Errorf("generate short id: %w", err)
		}

		storedShortURL, created, err = s.storage.Set(ctx, shortURL, longURL)
		if err != nil {
			if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "duplicate") {
				continue
			}
			return "", false, fmt.Errorf("save short url: %w", err)
		}
		break
	}

	if err != nil {
		return "", false, fmt.Errorf("save short url after retries: %w", err)
	}

	return strings.TrimSuffix(s.baseURL, "/") + "/" + storedShortURL, !created, nil
}

func (s *ShortenerService) ShortenBatch(ctx context.Context, items []model.ShortenBatchItem) ([]model.ShortenBatchResult, error) {
	pairs := make([]model.URLPair, len(items))
	for i, item := range items {
		shortID, err := generateID()
		if err != nil {
			return nil, err
		}
		pairs[i] = model.URLPair{ShortURL: shortID, LongURL: item.OriginalURL}
	}

	persistedPairs, err := s.storage.SetBatch(ctx, pairs)
	if err != nil {
		return nil, err
	}

	results := make([]model.ShortenBatchResult, len(items))
	for i, item := range items {
		results[i] = model.ShortenBatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      strings.TrimSuffix(s.baseURL, "/") + "/" + persistedPairs[i].ShortURL,
		}
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

func (s *ShortenerService) GetUserURLs(ctx context.Context) ([]model.URLPair, error) {
	urls, err := s.storage.GetUserURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user urls: %w", err)
	}

	results := make([]model.URLPair, len(urls))
	for i, url := range urls {
		results[i] = model.URLPair{
			ShortURL: strings.TrimSuffix(s.baseURL, "/") + "/" + url.ShortURL,
			LongURL:  url.LongURL,
		}
	}

	return results, nil
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}
