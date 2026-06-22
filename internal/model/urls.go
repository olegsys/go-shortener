package model

type URLPair struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

type ShortenBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenBatchResult struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
