package model

type URLStore struct {
	Urls map[string]string
}

func NewStore() *URLStore {
	return &URLStore{Urls: make(map[string]string)}
}
