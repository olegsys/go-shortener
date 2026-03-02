package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/olegsys/go-shortener/internal/model"
)

func TestHandler_Shorten(t *testing.T) {
	store := model.NewStore()
	h := &Handler{Store: store}

	url := "https://practicum.yandex.ru"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	w := httptest.NewRecorder()

	h.Shorten(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", res.StatusCode)
	}

	if len(store.Urls) == 0 {
		t.Error("Store is empty after Shorten request")
	}
}

func TestHandler_Redirect(t *testing.T) {
	store := model.NewStore()
	h := &Handler{Store: store}

	id := "testID"
	originalURL := "https://google.com"
	store.Urls[id] = originalURL

	req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
	w := httptest.NewRecorder()

	h.Redirect(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("Expected status 307, got %d", res.StatusCode)
	}

	location := res.Header.Get("Location")
	if location != originalURL {
		t.Errorf("Expected Location %s, got %s", originalURL, location)
	}
}
