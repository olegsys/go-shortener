package config

import (
	"flag"
	"os"
)

type Config struct {
	ListenAddress string
	BaseURL       string
}

func LoadConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", ":8080", "address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base url")
	flag.Parse()
	if val := os.Getenv("SERVER_ADDRESS"); val != "" {
		cfg.ListenAddress = val
	}
	if val := os.Getenv("BASE_URL"); val != "" {
		cfg.BaseURL = val
	}
	return cfg
}
