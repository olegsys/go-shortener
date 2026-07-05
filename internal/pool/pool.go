package pool

import (
	"sync"
)

// Pool - generic-структура, представляющая собой пул объектов
// Параметр T ограничен типами, которые реализуют метод Reset()
type Pool[T interface{ Reset() }] struct {
	items []T
	mu    sync.Mutex
}

// New - конструктор структуры Pool
func New[T interface{ Reset() }]() *Pool[T] {
	return &Pool[T]{
		items: make([]T, 0),
	}
}

// Get - метод, возвращающий объект из пула
func (p *Pool[T]) Get() T {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.items) == 0 {
		var zero T
		return zero
	}

	lastIdx := len(p.items) - 1
	item := p.items[lastIdx]

	var zero T
	p.items[lastIdx] = zero
	p.items = p.items[:lastIdx]

	return item
}

// Put - метод, помещающий объект обратно в пул
// Перед помещением вызывается метод Reset()
func (p *Pool[T]) Put(item T) {
	if any(item) != nil {
		item.Reset()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.items = append(p.items, item)
}
