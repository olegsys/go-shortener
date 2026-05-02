package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/model"
)

type Record struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}
type MapStorage struct {
	data     map[string]Record
	urlIndex map[string]map[string]string
	mutex    sync.RWMutex
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:     make(map[string]Record),
		urlIndex: make(map[string]map[string]string),
	}
}

func (m *MapStorage) Set(ctx context.Context, shortURL, longURL string) (string, bool, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.data[shortURL]; exists {
		return "", false, errors.New("short URL already exists")
	}

	userMap, ok := m.urlIndex[userID]
	if !ok {
		userMap = make(map[string]string)
		m.urlIndex[userID] = userMap
	}

	if existingShortURL, ok := userMap[longURL]; ok {
		return existingShortURL, false, nil
	}

	id := uuid.New()
	m.data[shortURL] = Record{
		UUID:        id.String(),
		ShortURL:    shortURL,
		OriginalURL: longURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	userMap[longURL] = shortURL

	return shortURL, true, nil
}

func (m *MapStorage) SetBatch(ctx context.Context, pairs []model.URLPair) ([]model.URLPair, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	results := make([]model.URLPair, len(pairs))

	userMap, ok := m.urlIndex[userID]
	if !ok {
		userMap = make(map[string]string)
		m.urlIndex[userID] = userMap
	}

	for i, pair := range pairs {
		shortURL := pair.ShortURL
		longURL := pair.LongURL

		if _, exists := m.data[shortURL]; exists {
			return nil, errors.New("short URL already exists: " + shortURL)
		}

		if existingShortID, ok := userMap[longURL]; ok {
			results[i] = model.URLPair{
				ShortURL: existingShortID,
				LongURL:  longURL,
			}
			continue
		}

		id := uuid.New()
		m.data[shortURL] = Record{
			UUID:        id.String(),
			ShortURL:    shortURL,
			OriginalURL: longURL,
			UserID:      userID,
			IsDeleted:   false,
		}
		userMap[longURL] = shortURL

		results[i] = model.URLPair{
			ShortURL: shortURL,
			LongURL:  longURL,
		}
	}

	return results, nil
}

func (m *MapStorage) Get(ctx context.Context, s string) (string, bool, bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	record, exist := m.data[s]
	if !exist {
		return "", false, false, nil
	}
	return record.OriginalURL, exist, record.IsDeleted, nil
}

func (m *MapStorage) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var records []Record
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, rec := range records {
		m.data[rec.ShortURL] = rec
		userMap, ok := m.urlIndex[rec.UserID]
		if !ok {
			userMap = make(map[string]string)
			m.urlIndex[rec.UserID] = userMap
		}
		userMap[rec.OriginalURL] = rec.ShortURL
	}
	return nil
}

func (m *MapStorage) SaveToFile(filePath string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	records := make([]Record, 0, len(m.data))
	for _, rec := range m.data {
		records = append(records, rec)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *MapStorage) GetUserURLs(ctx context.Context) ([]model.URLPair, error) {
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var urls []model.URLPair
	for _, rec := range m.data {
		if rec.UserID == userID && !rec.IsDeleted {
			urls = append(urls, model.URLPair{
				ShortURL: rec.ShortURL,
				LongURL:  rec.OriginalURL,
			})
		}
	}
	return urls, nil
}

func (m *MapStorage) DeleteBatch(ctx context.Context, userID string, ids []string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, id := range ids {
		if record, ok := m.data[id]; ok {
			if record.UserID == userID {
				record.IsDeleted = true
				m.data[id] = record
			}
		}
	}
	return nil
}
