package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/olegsys/go-shortener/internal/repository"
	"github.com/olegsys/go-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

type response struct {
	httpCode    int
	message     string
	contentType string
	checkBody   bool
}

func TestHandler_Shorten(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080/")
	h := NewHandler(svc)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://localhost:8080/", strings.NewReader(tt.inputBody))
			r.Header.Set("Content-Type", tt.contentType)
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
	}{"http://localhost:8080/", "abc", "https://yandex.ru/test"}

	storage := repository.NewMapStorage()
	err := storage.Set(context.Background(), mockData.id, mockData.longURL)
	assert.NoError(t, err)
	svc := service.NewShortenerService(storage, mockData.baseURL)
	h := NewHandler(svc)

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
	h := NewHandler(svc)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://localhost:8080/api/shorten", strings.NewReader(tt.inputBody))
			r.Header.Set("Content-Type", tt.contentType)
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
