package config

import "flag"

type Config struct {
	ListenAddress string
	BaseURL       string
}

func LoadConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", ":8080", "address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base url")
	flag.Parse()
	return cfg
}
