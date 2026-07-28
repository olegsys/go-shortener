// Package config отвечает за загрузку и хранение конфигурации приложения
package config

import (
	"flag"
	"os"
	"strconv"
)

// Config содержит все конфигурационные параметры приложения, загружаемые из флагов и переменных окружения
type Config struct {
	ListenAddress string
	BaseURL       string
	StorageFile   string
	DatabaseDSN   string
	SecretKey     string
	AuditFile     string
	AuditURL      string
	EnableHTTPS   bool
}

// LoadConfig загружает конфигурацию. Приоритет имеют переменные окружения, затем флаги командной строки
func LoadConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", ":8080", "address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base url")
	flag.StringVar(&cfg.StorageFile, "f", "", "storage file")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.StringVar(&cfg.SecretKey, "secret", "secret_key", "cookie secret key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit remote url")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
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
	if val := os.Getenv("SECRET_KEY"); val != "" {
		cfg.SecretKey = val
	}
	if val := os.Getenv("ENABLE_HTTPS"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.EnableHTTPS = b
		}
	}
	return cfg
}
