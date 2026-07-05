package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/middleware"
	"github.com/olegsys/go-shortener/internal/model"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

type mockDeletionService struct{}

func (m *mockDeletionService) Enqueue(userID, shortID string) error {
	return nil
}

type response struct {
	httpCode    int
	message     string
	contentType string
	checkBody   bool
}

func TestHandler_Shorten(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Post("/", h.Shorten)

	tests := []struct {
		name             string
		method           string
		inputBody        string
		contentType      string
		expectedResponse response
	}{
		{
			name:        "valid request",
			method:      http.MethodPost,
			inputBody:   "https://yandex.ru",
			contentType: "text/plain",
			expectedResponse: response{
				httpCode:    http.StatusCreated,
				message:     "http://localhost:8080/eJv4MRuJ",
				contentType: "text/plain",
				checkBody:   true,
			},
		},
		{
			name:        "invalid method request",
			method:      http.MethodGet,
			inputBody:   "https://yandex.ru",
			contentType: "text/plain",
			expectedResponse: response{
				httpCode:    http.StatusMethodNotAllowed,
				message:     "",
				contentType: "",
				checkBody:   false,
			},
		},
		{
			name:        "duplicate url returns conflict",
			method:      http.MethodPost,
			inputBody:   "https://yandex.ru",
			contentType: "text/plain",
			expectedResponse: response{
				httpCode:    http.StatusConflict,
				contentType: "text/plain",
				checkBody:   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://localhost:8080/", strings.NewReader(tt.inputBody))
			r.Header.Set("Content-Type", tt.contentType)
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "test-user")
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()
			shortURL, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedResponse.httpCode, res.StatusCode)
			if tt.expectedResponse.checkBody {
				assert.Regexp(t, regexp.MustCompile(`^http://localhost:8080/[^ /]{8}$`), string(shortURL))
			}
			if tt.expectedResponse.contentType != "" {
				assert.Equal(t, tt.expectedResponse.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandler_Redirect(t *testing.T) {
	mockData := struct {
		baseURL string
		id      string
		longURL string
		userID  string
	}{"http://localhost:8080/", "abc", "https://yandex.ru/test", "test-user-id"}

	storage := repository.NewMapStorage()
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, mockData.userID)
	_, _, err := storage.Set(ctx, mockData.userID, mockData.id, mockData.longURL)
	assert.NoError(t, err)
	svc := service.NewShortenerService(storage, mockData.baseURL)
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Get("/{id}", h.Redirect)

	tests := []struct {
		name         string
		method       string
		id           string
		wantStatus   int
		wantLocation string
	}{
		{
			name:         "success redirect",
			method:       http.MethodGet,
			id:           mockData.id,
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: mockData.longURL,
		},
		{
			name:       "url not found",
			id:         "fake",
			method:     http.MethodGet,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, mockData.baseURL+tt.id, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantLocation != "" {
				assert.Equal(t, tt.wantLocation, res.Header.Get("Location"))
			}

		})
	}
}

func TestHandler_ShortenJson(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Post("/api/shorten", h.ShortenJson)

	tests := []struct {
		name             string
		method           string
		inputBody        string
		contentType      string
		expectedResponse response
	}{
		{
			name:        "valid request",
			method:      http.MethodPost,
			inputBody:   `{"url":"https://practicum.yandex.ru"}`,
			contentType: "application/json",
			expectedResponse: response{
				httpCode:    http.StatusCreated,
				message:     `{"result":"http://localhost:8080/eJ4I8Mog"}`,
				contentType: "application/json",
				checkBody:   true,
			},
		},
		{
			name:        "invalid method request",
			method:      http.MethodGet,
			inputBody:   "https://yandex.ru",
			contentType: "application/json",
			expectedResponse: response{
				httpCode:    http.StatusMethodNotAllowed,
				message:     "",
				contentType: "",
				checkBody:   false,
			},
		},
		{
			name:        "invalid content type",
			method:      http.MethodPost,
			inputBody:   `{"url":"https://practicum.yandex.ru"}`,
			contentType: "text/plain",
			expectedResponse: response{
				httpCode:    http.StatusBadRequest,
				message:     "",
				contentType: "",
				checkBody:   false,
			},
		},
		{
			name:        "duplicate url returns conflict",
			method:      http.MethodPost,
			inputBody:   `{"url":"https://practicum.yandex.ru"}`,
			contentType: "application/json",
			expectedResponse: response{
				httpCode:    http.StatusConflict,
				contentType: "application/json",
				checkBody:   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://localhost:8080/api/shorten", strings.NewReader(tt.inputBody))
			r.Header.Set("Content-Type", tt.contentType)
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "test-user")
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()
			shortURL, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResponse.httpCode, res.StatusCode)
			if tt.expectedResponse.checkBody {
				assert.Regexp(t, regexp.MustCompile(`\{"result":"http://localhost:8080/([^"]{8})"\}`), string(shortURL))
			}
			if tt.expectedResponse.contentType != "" {
				assert.Equal(t, tt.expectedResponse.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandler_ShortenBatchJson(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Post("/api/shorten/batch", h.ShortenBatchJson)

	tests := []struct {
		name        string
		method      string
		inputBody   string
		contentType string
		wantStatus  int
		wantCount   int
	}{
		{
			name:        "valid batch request",
			method:      http.MethodPost,
			inputBody:   `[{"correlation_id":"1","original_url":"https://practicum.yandex.ru"},{"correlation_id":"2","original_url":"https://ya.ru"}]`,
			contentType: "application/json",
			wantStatus:  http.StatusCreated,
			wantCount:   2,
		},
		{
			name:        "empty batch request",
			method:      http.MethodPost,
			inputBody:   `[]`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCount:   0,
		},
		{
			name:        "invalid content type",
			method:      http.MethodPost,
			inputBody:   `[{"correlation_id":"1","original_url":"https://practicum.yandex.ru"}]`,
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://localhost:8080/api/shorten/batch", strings.NewReader(tt.inputBody))
			r.Header.Set("Content-Type", tt.contentType)
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "test-user")
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantStatus == http.StatusCreated {
				var batchResp []model.ShortenBatchResult
				err := json.NewDecoder(res.Body).Decode(&batchResp)
				assert.NoError(t, err)
				assert.Len(t, batchResp, tt.wantCount)
				for _, item := range batchResp {
					assert.NotEmpty(t, item.CorrelationID)
					assert.Regexp(t, regexp.MustCompile(`^http://localhost:8080/[^ /]{8}$`), item.ShortURL)
				}
				assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandler_Shorten_DuplicateReturnsExistingURL(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Post("/", h.Shorten)

	firstReq := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader("https://example.com"))
	firstReq.Header.Set("Content-Type", "text/plain")
	ctx := context.WithValue(firstReq.Context(), middleware.UserIDKey, "test-user")
	firstReq = firstReq.WithContext(ctx)
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)
	assert.Equal(t, http.StatusCreated, firstResp.Code)

	firstBody, err := io.ReadAll(firstResp.Result().Body)
	assert.NoError(t, err)

	secondReq := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader("https://example.com"))
	secondReq.Header.Set("Content-Type", "text/plain")
	ctx2 := context.WithValue(secondReq.Context(), middleware.UserIDKey, "test-user")
	secondReq = secondReq.WithContext(ctx2)
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)
	assert.Equal(t, http.StatusConflict, secondResp.Code)

	secondBody, err := io.ReadAll(secondResp.Result().Body)
	assert.NoError(t, err)
	assert.Equal(t, string(firstBody), string(secondBody))
}

func TestHandler_ShortenJSON_DuplicateReturnsExistingURL(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	router := chi.NewRouter()
	router.Post("/api/shorten", h.ShortenJson)

	firstReq := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(firstReq.Context(), middleware.UserIDKey, "test-user")
	firstReq = firstReq.WithContext(ctx)
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)
	assert.Equal(t, http.StatusCreated, firstResp.Code)

	var firstPayload resp
	err := json.NewDecoder(firstResp.Result().Body).Decode(&firstPayload)
	assert.NoError(t, err)

	secondReq := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	ctx2 := context.WithValue(secondReq.Context(), middleware.UserIDKey, "test-user")
	secondReq = secondReq.WithContext(ctx2)
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)
	assert.Equal(t, http.StatusConflict, secondResp.Code)

	var secondPayload resp
	err = json.NewDecoder(secondResp.Result().Body).Decode(&secondPayload)
	assert.NoError(t, err)
	assert.Equal(t, firstPayload.Result, secondPayload.Result)
}

func TestHandler_GetUserURLs(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc, &mockDeletionService{}, nil)

	userID := "test-user-123"
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
	_, _, err := storage.Set(ctx, userID, "urltest1", "https://example.com")
	assert.NoError(t, err)
	_, _, err = storage.Set(ctx, userID, "urltest2", "https://yandex.ru")
	assert.NoError(t, err)

	router := chi.NewRouter()
	router.Get("/api/user/urls", h.GetUserURLs)

	tests := []struct {
		name       string
		userID     string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "success with URLs",
			userID:     userID,
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "no URLs returns 204",
			userID:     "empty-user",
			wantStatus: http.StatusNoContent,
			wantCount:  0,
		},
		{
			name:       "no userID returns 401",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
			if tt.userID != "" {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, tt.userID)
				r = r.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantStatus == http.StatusOK {
				var urls []model.URLPair
				err := json.NewDecoder(res.Body).Decode(&urls)
				assert.NoError(t, err)
				assert.Len(t, urls, tt.wantCount)
				for _, url := range urls {
					assert.NotEmpty(t, url.ShortURL)
					assert.NotEmpty(t, url.LongURL)
					assert.Regexp(t, regexp.MustCompile(`^http://localhost:8080/`), url.ShortURL)
				}
				assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
			}
		})
	}
}
