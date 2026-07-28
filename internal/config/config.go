// Package config отвечает за загрузку и хранение конфигурации приложения
package config

import (
	"encoding/json"
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
	ConfigFile    string
}

// fileConfig описывает структуру json файла конфигурации.
type fileConfig struct {
	Address     string `json:"address"`
	BaseURL     string `json:"base_url"`
	StoreFile   string `json:"store_file"`
	DatabaseDSN string `json:"database_dsn"`
	SecretKey   string `json:"secret_key"`
	AuditFile   string `json:"audit_file"`
	AuditURL    string `json:"audit_url"`
	EnableHTTPS *bool  `json:"enable_https"`
}

// LoadConfig загружает конфигурацию. Приоритет имеют переменные окружения, затем флаги командной строки, затем json файл
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
	flag.StringVar(&cfg.ConfigFile, "c", "", "path to config file")
	flag.Parse()

	// Определяем, какие флаги были заданы
	explicitFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	if !explicitFlags["c"] {
		if envPath := os.Getenv("CONFIG"); envPath != "" {
			cfg.ConfigFile = envPath
		}
	}

	var fc fileConfig
	if cfg.ConfigFile != "" {
		data, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			println("warning: cannot read config file:", err.Error())
		} else if err := json.Unmarshal(data, &fc); err != nil {
			println("warning: cannot parse config file:", err.Error())
		}
	}
	applyFileConfig(cfg, &fc)
	applyEnvConfig(cfg)
	applyExplicitFlags(cfg, explicitFlags)

	return cfg
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
	if val := os.Getenv("CONFIG"); val != "" {
		cfg.ConfigFile = val
	}
}

// applyExplicitFlags применяет значения явно заданных флагов через cli
func applyExplicitFlags(cfg *Config, explicit map[string]bool) {
	if explicit["a"] {
		cfg.ListenAddress = flag.Lookup("a").Value.String()
	}
	if explicit["b"] {
		cfg.BaseURL = flag.Lookup("b").Value.String()
	}
	if explicit["f"] {
		cfg.StorageFile = flag.Lookup("f").Value.String()
	}
	if explicit["d"] {
		cfg.DatabaseDSN = flag.Lookup("d").Value.String()
	}
	if explicit["secret"] {
		cfg.SecretKey = flag.Lookup("secret").Value.String()
	}
	if explicit["audit-file"] {
		cfg.AuditFile = flag.Lookup("audit-file").Value.String()
	}
	if explicit["audit-url"] {
		cfg.AuditURL = flag.Lookup("audit-url").Value.String()
	}
	if explicit["s"] {
		cfg.EnableHTTPS, _ = strconv.ParseBool(flag.Lookup("s").Value.String())
	}
	if explicit["c"] {
		cfg.ConfigFile = flag.Lookup("c").Value.String()
		if cfg.ConfigFile == "" {
			cfg.ConfigFile = flag.Lookup("config").Value.String()
		}
	}
}
