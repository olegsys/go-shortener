package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type HTTPObserver struct {
	url    string
	client *http.Client
	ch     chan Event
	wg     sync.WaitGroup
}

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
			resp.Body.Close()
		}
	}
}

func (ho *HTTPObserver) Update(ctx context.Context, event Event) {
	select {
	case ho.ch <- event:
	default:
		// Если канал переполнен, отбрасываем событие, чтобы не блокировать основной поток
	}
}

func (ho *HTTPObserver) Close() error {
	close(ho.ch)
	ho.wg.Wait() // Ждём, пока все события из канала будут отправлены
	return nil
}
