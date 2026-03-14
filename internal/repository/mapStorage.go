package repository

import (
	"sync"
)

type MapStorage struct {
	data  map[string]string
	mutex sync.RWMutex
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data: make(map[string]string),
	}
}

func (m *MapStorage) Set(shortURL, longURL string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[shortURL] = longURL
}

func (m *MapStorage) Get(s string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	longURL, exist := m.data[s]
	return longURL, exist
}
