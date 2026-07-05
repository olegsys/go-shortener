package service

import (
	"context"
	"testing"

	"github.com/olegsys/go-shortener/internal/model"
	"github.com/olegsys/go-shortener/internal/repository"
)

func BenchmarkShorten(b *testing.B) {
	storage := repository.NewMapStorage()
	svc := NewShortenerService(storage, "http://localhost:8080/")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = svc.Shorten(ctx, "https://example.com/some/long/url/that/needs/shortening")
	}
}

func BenchmarkShortenBatch(b *testing.B) {
	storage := repository.NewMapStorage()
	svc := NewShortenerService(storage, "http://localhost:8080/")
	ctx := context.Background()

	items := make([]model.ShortenBatchItem, 100)
	for i := 0; i < 100; i++ {
		items[i] = model.ShortenBatchItem{
			CorrelationID: string(rune(i)),
			OriginalURL:   "https://example.com/some/long/url/that/needs/shortening",
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ShortenBatch(ctx, items)
	}
}

func BenchmarkResolve(b *testing.B) {
	storage := repository.NewMapStorage()
	svc := NewShortenerService(storage, "http://localhost:8080/")
	ctx := context.Background()

	id := "testid12"
	storage.Set(ctx, "user1", id, "https://example.com")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = svc.Resolve(ctx, id)
	}
}
