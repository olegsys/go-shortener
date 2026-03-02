package main

import (
	"net/http"

	"github.com/olegsys/go-shortener/internal/handler"
	"github.com/olegsys/go-shortener/internal/model"
)

func main() {
	store := model.NewStore()
	h := &handler.Handler{
		Store: store,
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Shorten(w, r)
		case http.MethodGet:
			h.Redirect(w, r)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	err := http.ListenAndServe(`:8080`, nil)
	if err != nil {
		panic(err)
	}
}
