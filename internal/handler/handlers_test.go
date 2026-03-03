package handler

import (
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

func TestShorten(t *testing.T) {
	storage := repository.NewMapStorage()
	svc := service.NewShortenerService(storage, "http://localhost:8080")
	h := NewHandler(svc)

	router := chi.NewRouter()
	router.Post("/", h.Shorten)

	type response struct {
		httpCode    int
		message     string
		contentType string
		checkBody   bool
	}
	tests := []struct {
		name             string
		method           string
		inputBody        string
		contentType      string
		expectedResponse struct {
			httpCode    int
			message     string
			contentType string
			checkBody   bool
		}
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
			name:        "invalide method request",
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

			assert.Equal(t, tt.expectedResponse.httpCode, w.Result().StatusCode)
			if tt.expectedResponse.checkBody {
				assert.Regexp(t, regexp.MustCompile(`^http://localhost:8080/[^ /]{8}$`), w.Body.String())
			}
			if tt.expectedResponse.contentType != "" {
				assert.Equal(t, tt.expectedResponse.contentType, w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestRedirect(t *testing.T) {

}
