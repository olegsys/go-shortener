// Package repository предоставляет реализации хранилищ данных
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/olegsys/go-shortener/internal/model"
)

// Record представляет собой одну запись в хранилище URL
type Record struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

type userURLKey struct {
	userID  string
	longURL string
}

// MapStorage реализует интерфейс URLStore, храня данные в оперативной памяти с возможностью сохранения в файл
type MapStorage struct {
	data     map[string]Record
	urlIndex map[userURLKey]string
	mutex    sync.RWMutex
}

// NewMapStorage создает и инициализирует новое in-memory хранилище
func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:     make(map[string]Record),
		urlIndex: make(map[userURLKey]string),
	}
}

// Set сохраняет пару shortURL и longURL для пользователя userID
func (m *MapStorage) Set(ctx context.Context, userID, shortURL, longURL string) (string, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.data[shortURL]; exists {
		return "", false, errors.New("short URL already exists")
	}

	key := userURLKey{userID: userID, longURL: longURL}
	if existingShortURL, ok := m.urlIndex[key]; ok {
		return existingShortURL, false, nil
	}

	m.data[shortURL] = Record{
		ShortURL:    shortURL,
		OriginalURL: longURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	m.urlIndex[key] = shortURL

	return shortURL, true, nil
}

// SetBatch сохраняет набор пар URL для пользователя userID
func (m *MapStorage) SetBatch(ctx context.Context, userID string, pairs []model.URLPair) ([]model.URLPair, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	results := make([]model.URLPair, len(pairs))

	for i, pair := range pairs {
		shortURL := pair.ShortURL
		longURL := pair.LongURL

		if _, exists := m.data[shortURL]; exists {
			return nil, errors.New("short URL already exists: " + shortURL)
		}

		key := userURLKey{userID: userID, longURL: longURL}
		if existingShortID, ok := m.urlIndex[key]; ok {
			results[i] = model.URLPair{ShortURL: existingShortID, LongURL: longURL}
			continue
		}

		m.data[shortURL] = Record{
			ShortURL:    shortURL,
			OriginalURL: longURL,
			UserID:      userID,
			IsDeleted:   false,
		}
		m.urlIndex[key] = shortURL

		results[i] = model.URLPair{ShortURL: shortURL, LongURL: longURL}
	}

	return results, nil
}

// Get возвращает оригинальный URL и статусы существования,удаления по короткому URL
func (m *MapStorage) Get(ctx context.Context, s string) (string, bool, bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	record, exist := m.data[s]
	if !exist {
		return "", false, false, nil
	}
	return record.OriginalURL, exist, record.IsDeleted, nil
}

// LoadFromFile загружает данные хранилища из JSON-файла
func (m *MapStorage) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = file.Close() }()

	var records []Record
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, rec := range records {
		m.data[rec.ShortURL] = rec
		key := userURLKey{userID: rec.UserID, longURL: rec.OriginalURL}
		m.urlIndex[key] = rec.ShortURL
	}
	return nil
}

// SaveToFile сохраняет текущее состояние хранилища в JSON-файл
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

// GetUserURLs возвращает все URL, принадлежащие пользователю userID
func (m *MapStorage) GetUserURLs(ctx context.Context, userID string) ([]model.URLPair, error) {
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

// DeleteBatch помечает указанные URL как удаленные для пользователя userID
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

// Stats возвращает количество сокращённых URL и количество уникальных пользователей
func (m *MapStorage) Stats(ctx context.Context) (int, int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	urls := len(m.data)

	users := make(map[string]struct{})
	for _, rec := range m.data {
		users[rec.UserID] = struct{}{}
	}

	return urls, len(users), nil
}
