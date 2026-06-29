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
	ch   chan Event
	wg   sync.WaitGroup
}

// NewFileObserver создает новый FileObserver, открывая или создавая файл по указанному пути,
// и запускает фоновый воркер для безопасной записи.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	fo := &FileObserver{
		file: f,
		ch:   make(chan Event, 1000),
	}

	fo.wg.Add(1)
	go fo.worker()

	return fo, nil
}

// worker читает события из канала и последовательно записывает их в файл
func (fo *FileObserver) worker() {
	defer fo.wg.Done()
	for event := range fo.ch {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		_, _ = fo.file.Write(data)
		_, _ = fo.file.Write([]byte("\n"))
	}
}

// Update помещает событие в канал для асинхронной записи.
// Если канал переполнен, событие отбрасывается, чтобы не блокировать основной поток (HTTP-запрос).
func (fo *FileObserver) Update(ctx context.Context, event Event) {
	select {
	case fo.ch <- event:
	default:
		// Канал переполнен, отбрасываем событие
	}
}

// Close закрывает канал, ожидает завершения записи всех накопленных событий и закрывает файл
func (fo *FileObserver) Close() error {
	close(fo.ch)
	fo.wg.Wait()
	return fo.file.Close()
}
