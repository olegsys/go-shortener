// Package config отвечает за загрузку и хранение конфигурации приложения
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config содержит все конфигурационные параметры приложения, загружаемые из флагов и переменных окружения
type Config struct {
	ListenAddress     string
	BaseURL           string
	StorageFile       string
	DatabaseDSN       string
	SecretKey         string
	AuditFile         string
	AuditURL          string
	EnableHTTPS       bool
	ConfigFile        string
	CertFile          string
	KeyFile           string
	TrustedSubnet     string
	GRPCListenAddress string
}

// fileConfig описывает структуру json файла конфигурации.
type fileConfig struct {
	Address       string `json:"address"`
	BaseURL       string `json:"base_url"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	SecretKey     string `json:"secret_key"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
	EnableHTTPS   *bool  `json:"enable_https"`
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	TrustedSubnet string `json:"trusted_subnet"`
	GRPCAddress   string `json:"grpc_address"`
}

// LoadConfig загружает конфигурацию. Приоритет имеют переменные окружения, затем флаги командной строки, затем json файл
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddress, "a", ":8080", "address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base url")
	flag.StringVar(&cfg.StorageFile, "f", "", "storage file")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.StringVar(&cfg.SecretKey, "secret", "secret_key", "cookie secret key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit remote url")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&cfg.ConfigFile, "c", "", "path to config file")
	flag.StringVar(&cfg.CertFile, "cert-file", "cert.pem", "path to TLS certificate file")
	flag.StringVar(&cfg.KeyFile, "key-file", "key.pem", "path to TLS private key file")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "trusted subnet in CIDR notation")
	flag.StringVar(&cfg.GRPCListenAddress, "grpc-a", "", "gRPC listen address")
	flag.Parse()

	var configFlagSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "c" {
			configFlagSet = true
		}
	})
	if !configFlagSet {
		if envPath := os.Getenv("CONFIG"); envPath != "" {
			cfg.ConfigFile = envPath
		}
	}

	var fc fileConfig
	if cfg.ConfigFile != "" {
		data, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("read config file %q: %w", cfg.ConfigFile, err)
		}

		if err := json.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("parse config file %q: %w", cfg.ConfigFile, err)
		}
	}

	applyFileConfig(cfg, &fc)
	applyEnvConfig(cfg)
	applyExplicitFlags(cfg)

	return cfg, nil
}

// applyFileConfig применяет значения из json файла конфигурации
func applyFileConfig(cfg *Config, fc *fileConfig) {
	if fc.Address != "" {
		cfg.ListenAddress = fc.Address
	}
	if fc.BaseURL != "" {
		cfg.BaseURL = fc.BaseURL
	}
	if fc.StoreFile != "" {
		cfg.StorageFile = fc.StoreFile
	}
	if fc.DatabaseDSN != "" {
		cfg.DatabaseDSN = fc.DatabaseDSN
	}
	if fc.SecretKey != "" {
		cfg.SecretKey = fc.SecretKey
	}
	if fc.AuditFile != "" {
		cfg.AuditFile = fc.AuditFile
	}
	if fc.AuditURL != "" {
		cfg.AuditURL = fc.AuditURL
	}
	if fc.EnableHTTPS != nil {
		cfg.EnableHTTPS = *fc.EnableHTTPS
	}
	if fc.CertFile != "" {
		cfg.CertFile = fc.CertFile
	}
	if fc.KeyFile != "" {
		cfg.KeyFile = fc.KeyFile
	}
	if fc.TrustedSubnet != "" {
		cfg.TrustedSubnet = fc.TrustedSubnet
	}
	if fc.GRPCAddress != "" {
		cfg.GRPCListenAddress = fc.GRPCAddress
	}
}

// applyEnvConfig применяет значения из переменных окружения
func applyEnvConfig(cfg *Config) {
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
	if val := os.Getenv("CERT_FILE"); val != "" {
		cfg.CertFile = val
	}
	if val := os.Getenv("KEY_FILE"); val != "" {
		cfg.KeyFile = val
	}
	if val := os.Getenv("TRUSTED_SUBNET"); val != "" {
		cfg.TrustedSubnet = val
	}
	if val := os.Getenv("GRPC_ADDRESS"); val != "" {
		cfg.GRPCListenAddress = val
	}
}

// applyExplicitFlags применяет значения явно заданных флагов через cli
func applyExplicitFlags(cfg *Config) {
	stringFlags := map[string]*string{
		"a":          &cfg.ListenAddress,
		"b":          &cfg.BaseURL,
		"f":          &cfg.StorageFile,
		"d":          &cfg.DatabaseDSN,
		"secret":     &cfg.SecretKey,
		"audit-file": &cfg.AuditFile,
		"audit-url":  &cfg.AuditURL,
		"c":          &cfg.ConfigFile,
		"cert-file":  &cfg.CertFile,
		"key-file":   &cfg.KeyFile,
		"t":          &cfg.TrustedSubnet,
		"grpc-a":     &cfg.GRPCListenAddress,
	}

	boolFlags := map[string]*bool{
		"s": &cfg.EnableHTTPS,
	}

	flag.Visit(func(f *flag.Flag) {
		if dst, ok := stringFlags[f.Name]; ok {
			*dst = f.Value.String()
			return
		}

		if dst, ok := boolFlags[f.Name]; ok {
			*dst, _ = strconv.ParseBool(f.Value.String())
			return
		}
	})
}
