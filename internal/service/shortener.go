package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/model"
)

// URLStore описывает интерфейс хранилища для работы с URL
type URLStore interface {
	Set(ctx context.Context, userID, shortURL, longURL string) (string, bool, error)
	SetBatch(ctx context.Context, userID string, pairs []model.URLPair) ([]model.URLPair, error)
	Get(ctx context.Context, shortURL string) (string, bool, bool, error)
	GetUserURLs(ctx context.Context, userID string) ([]model.URLPair, error)
	DeleteBatch(ctx context.Context, userID string, ids []string) error
}

// ShortenerService содержит бизнес-логику сервиса сокращения URL
type ShortenerService struct {
	storage URLStore
	baseURL string
}

// NewShortenerService создает и инициализирует новый экземпляр ShortenerService
func NewShortenerService(storage URLStore, baseURL string) *ShortenerService {
	return &ShortenerService{
		storage: storage,
		baseURL: baseURL,
	}
}

// Shorten генерирует короткий URL для переданного longURL
func (s *ShortenerService) Shorten(ctx context.Context, longURL string) (string, bool, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)
	const maxRetries = 10
	var storedShortURL string
	var created bool
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		shortURL, err := generateID()
		if err != nil {
			return "", false, fmt.Errorf("generate short id: %w", err)
		}

		storedShortURL, created, err = s.storage.Set(ctx, userID, shortURL, longURL)
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

// ShortenBatch генерирует короткие URL для батча оригинальных URL
func (s *ShortenerService) ShortenBatch(ctx context.Context, items []model.ShortenBatchItem) ([]model.ShortenBatchResult, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	pairs := make([]model.URLPair, len(items))
	for i, item := range items {
		shortID, err := generateID()
		if err != nil {
			return nil, err
		}
		pairs[i] = model.URLPair{ShortURL: shortID, LongURL: item.OriginalURL}
	}

	persistedPairs, err := s.storage.SetBatch(ctx, userID, pairs)
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

// Resolve возвращает оригинальный URL по его короткому идентификатору
func (s *ShortenerService) Resolve(ctx context.Context, shortURL string) (string, bool, bool, error) {
	longURL, exists, isDeleted, err := s.storage.Get(ctx, shortURL)
	if err != nil {
		return "", false, false, fmt.Errorf("resolve short url: %w", err)
	}
	return longURL, exists, isDeleted, nil
}

// GetUserURLs возвращает все URL, принадлежащие текущему пользователю из контекста
func (s *ShortenerService) GetUserURLs(ctx context.Context) ([]model.URLPair, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)
	urls, err := s.storage.GetUserURLs(ctx, userID)
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

// DeleteURLs помечает указанные URL как удаленные
func (s *ShortenerService) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	return s.storage.DeleteBatch(ctx, userID, ids)
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}
