package main

import (
	"errors"
	"os"
	"time"
)

type config struct {
	Address                 string
	DBPath                  string
	TokenTTL                time.Duration
	AdminEmail              string
	AdminPassword           string
	OperatorEmail           string
	OperatorPassword        string
	UsingDefaultCredentials bool
}

func loadConfig() (config, error) {
	cfg := config{
		Address:          envOrDefault("APP_ADDR", ":8080"),
		DBPath:           envOrDefault("APP_DB_PATH", "./data/customer-registry.db"),
		AdminEmail:       envOrDefault("APP_ADMIN_EMAIL", "admin@example.local"),
		AdminPassword:    envOrDefault("APP_ADMIN_PASSWORD", "Admin123!"),
		OperatorEmail:    envOrDefault("APP_OPERATOR_EMAIL", "operator@example.local"),
		OperatorPassword: envOrDefault("APP_OPERATOR_PASSWORD", "Operator123!"),
	}

	ttl, err := time.ParseDuration(envOrDefault("APP_TOKEN_TTL", "24h"))
	if err != nil || ttl <= 0 {
		return config{}, errors.New("APP_TOKEN_TTL deve essere una durata positiva, per esempio 24h")
	}
	cfg.TokenTTL = ttl
	cfg.UsingDefaultCredentials = os.Getenv("APP_ADMIN_PASSWORD") == "" || os.Getenv("APP_OPERATOR_PASSWORD") == ""
	return cfg, nil
}

func envOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}
