package repository

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/olegsys/go-shortener/internal/model"
)

type Record struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
type MapStorage struct {
	data     map[string]Record
	urlIndex map[string]string
	mutex    sync.RWMutex
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:     make(map[string]Record),
		urlIndex: make(map[string]string),
	}
}

func (m *MapStorage) Set(ctx context.Context, shortURL, longURL string) (string, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if existingShortURL, ok := m.urlIndex[longURL]; ok {
		return existingShortURL, false, nil
	}

	id := uuid.New()
	m.data[shortURL] = Record{
		UUID:        id.String(),
		ShortURL:    shortURL,
		OriginalURL: longURL,
	}
	m.urlIndex[longURL] = shortURL

	return shortURL, true, nil
}

func (m *MapStorage) SetBatch(ctx context.Context, pairs []model.URLPair) ([]model.URLPair, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	results := make([]model.URLPair, len(pairs))
	for i, pair := range pairs {
		if existingShortID, ok := m.urlIndex[pair.LongURL]; ok {
			results[i] = model.URLPair{
				ShortURL: existingShortID,
				LongURL:  pair.LongURL,
			}
			continue
		}

		id := uuid.New()
		m.data[pair.ShortURL] = Record{
			UUID:        id.String(),
			ShortURL:    pair.ShortURL,
			OriginalURL: pair.LongURL,
		}
		m.urlIndex[pair.LongURL] = pair.ShortURL

		results[i] = pair
	}

	return results, nil
}

func (m *MapStorage) Get(ctx context.Context, s string) (string, bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	record, exist := m.data[s]
	return record.OriginalURL, exist, nil
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
		m.urlIndex[rec.OriginalURL] = rec.ShortURL
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
