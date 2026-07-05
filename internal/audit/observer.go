package audit

import (
	"context"
	"sync"
)

// Auditor - интерфейс для публикации событий
type Auditor interface {
	Publish(ctx context.Context, event Event)
}

// Observer - интерфейс для всех приёмников событий аудита
type Observer interface {
	Update(ctx context.Context, event Event)
	Close() error
}

// EventBus - хранит список наблюдателей и рассылает им события
type EventBus struct {
	observers []Observer
	mu        sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

// Register добавляет наблюдателя
func (eb *EventBus) Register(obs Observer) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.observers = append(eb.observers, obs)
}

// Publish рассылает событие всем зарегистрированным наблюдателям
func (eb *EventBus) Publish(ctx context.Context, event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Используем background контекст, чтобы аудит не прервался при отключении клиента
	bgCtx := context.Background()
	for _, obs := range eb.observers {
		obs.Update(bgCtx, event)
	}
}

// Close закрывает всех наблюдателей
func (eb *EventBus) Close() {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, obs := range eb.observers {
		obs.Close()
	}
}
