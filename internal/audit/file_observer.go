package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

// FileObserver реализует интерфейс Observer, записывая события аудита в файл
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создает новый FileObserver, открывая или создавая файл по указанному пути
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{file: f}, nil
}

// Update записывает событие аудита в файл в формате JSON
func (fo *FileObserver) Update(ctx context.Context, event Event) {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fo.file.Write(data)
	fo.file.Write([]byte("\n"))
}

// Close закрывает файл, связанный с FileObserver
func (fo *FileObserver) Close() error {
	return fo.file.Close()
}
