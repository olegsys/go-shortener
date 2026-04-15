package repository

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"
)

type Record struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
type MapStorage struct {
	data  map[string]Record
	mutex sync.RWMutex
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data: make(map[string]Record),
	}
}

func (m *MapStorage) Set(ctx context.Context, shortURL, longURL string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	id := uuid.New()
	m.data[shortURL] = Record{
		UUID:        id.String(),
		ShortURL:    shortURL,
		OriginalURL: longURL,
	}
	return nil
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
