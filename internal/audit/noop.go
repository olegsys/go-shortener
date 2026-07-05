package audit

import "context"

// NoopAuditor реализует паттерн Null Object.
// Используется как заглушка, когда аудит не сконфигурирован
type NoopAuditor struct{}

// Publish ничего не делает, игнорируя событие
func (NoopAuditor) Publish(ctx context.Context, event Event) {}
