package config

import (
	"flag"
	"os"
)

type Config struct {
	ListenAddress string
	BaseURL       string
	StorageFile   string
	DatabaseDSN   string
}

func LoadConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", ":8080", "address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base url")
	flag.StringVar(&cfg.StorageFile, "f", "storage.json", "storage file")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.Parse()
	if val := os.Getenv("SERVER_ADDRESS"); val != "" {
		cfg.ListenAddress = val
	}
	if val := os.Getenv("BASE_URL"); val != "" {
		cfg.BaseURL = val
	}
	if val := os.Getenv("FILE_STORAGE_PATH"); val != "" {
		cfg.StorageFile = val
	}
	if val := os.Getenv("DATABASE_DSN"); val != "" {
		cfg.DatabaseDSN = val
	}
	return cfg
}
