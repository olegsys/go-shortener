package config

import "flag"

type Config struct {
	ListenAddress string
	BaseUrl       string
}

func LoadConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", "localhost:8080", "address")
	flag.StringVar(&cfg.BaseUrl, "b", "http://localhost:8080/", "base url")
	flag.Parse()
	return cfg
}
