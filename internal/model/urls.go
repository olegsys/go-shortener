// Package model описывает доменные структуры данных сервиса
package model

// URLPair представляет собой пару сокращенного и оригинального URL
type URLPair struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

// ShortenBatchItem описывает один элемент в запросе на пакетном сокращении URL
type ShortenBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// ShortenBatchResult описывает результат пакетного сокращения URL
type ShortenBatchResult struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
