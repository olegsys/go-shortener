package audit

// Event представляет собой событие аудита для логирования действий пользователей
type Event struct {
	Ts     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id"`
	URL    string `json:"url"`
}
