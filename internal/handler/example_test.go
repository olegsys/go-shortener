package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
)

// setupTestRouter создает тестовый роутер и хендлеры для примеров.
func setupTestRouter() *chi.Mux {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080")
	deletionSvc := &mockDeletionService{} // mockDeletionService уже определен в handlers_test.go

	h := NewHandler(svc, deletionSvc, nil)
	router := chi.NewRouter()

	// Имитируем middleware авторизации, добавляя userID в контекст
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "example-user-id")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	router.Post("/", h.Shorten)
	router.Post("/api/shorten", h.ShortenJSON)
	router.Get("/{id}", h.Redirect)

	return router
}

// ExampleHandler_Shorten демонстрирует сокращение URL через POST / с Content-Type: text/plain.
func ExampleHandler_Shorten() {
	router := setupTestRouter()

	// Создаем запрос
	body := strings.NewReader("https://yandex.ru")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	// Выполняем запрос
	router.ServeHTTP(w, req)
	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	// Выводим статус код для проверки в // Output:
	fmt.Println(res.StatusCode)

	// Output: 201
}

// ExampleHandler_ShortenJson демонстрирует сокращение URL через POST /api/shorten с JSON.
func ExampleHandler_ShortenJSON() {
	router := setupTestRouter()

	// Создаем JSON запрос
	jsonBody := `{"url": "https://practicum.yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	fmt.Println(res.StatusCode)
	fmt.Println(res.Header.Get("Content-Type"))

	// Output:
	// 201
	// application/json
}

// ExampleHandler_Redirect демонстрирует редирект по короткому URL.
func ExampleHandler_Redirect() {
	// Сначала сохраним URL, чтобы было куда редиректить
	storage := repository.NewMapStorage()
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "example-user-id")
	_, _, _ = storage.Set(ctx, "example-user-id", "abc12345", "https://ya.ru")

	svc := service.NewShortenerService(storage, "http://localhost:8080")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Get("/{id}", h.Redirect)

	// Запрашиваем редирект
	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	fmt.Println(res.StatusCode)
	fmt.Println(res.Header.Get("Location"))

	// Output:
	// 307
	// https://ya.ru
}
