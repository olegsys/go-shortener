package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HTTPObserver реализует интерфейс Observer, отправляя события аудита на удаленный HTTP-сервер
type HTTPObserver struct {
	url    string
	client *http.Client
	ch     chan Event
	wg     sync.WaitGroup
}

// NewHTTPObserver создает новый HTTPObserver и запускает фоновый воркер для отправки событий
func NewHTTPObserver(url string) *HTTPObserver {
	obs := &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
		ch:     make(chan Event, 1000),
	}
	obs.wg.Add(1)
	go obs.worker()
	return obs
}

// worker читает события из канала и отправляет их на HTTP endpoint
func (ho *HTTPObserver) worker() {
	defer ho.wg.Done()
	for event := range ho.ch {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		req, err := http.NewRequest(http.MethodPost, ho.url, bytes.NewReader(data))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := ho.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
}

// Update помещает событие в канал для асинхронной отправки. Если канал переполнен, событие отбрасывается
func (ho *HTTPObserver) Update(ctx context.Context, event Event) {
	select {
	case ho.ch <- event:
	default:
		// Если канал переполнен, отбрасываем событие, чтобы не блокировать основной поток
	}
}

// Close закрывает канал и ожидает завершения отправки всех накопленных событий
func (ho *HTTPObserver) Close() error {
	close(ho.ch)
	ho.wg.Wait()
	return nil
}
