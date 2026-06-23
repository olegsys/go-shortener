package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{file: f}, nil
}

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

func (fo *FileObserver) Close() error {
	return fo.file.Close()
}
