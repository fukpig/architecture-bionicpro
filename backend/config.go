package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	KeycloakURL    string
	KeycloakRealm  string
	ClickHouseHost string
	ClickHousePort string
	ClickHouseDB   string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	return &Config{
		Port:           getEnv("PORT", "8000"),
		KeycloakURL:    getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		KeycloakRealm:  getEnv("KEYCLOAK_REALM", "reports-realm"),
		ClickHouseHost: getEnv("CLICKHOUSE_HOST", "localhost"),
		ClickHousePort: getEnv("CLICKHOUSE_PORT", "9000"),
		ClickHouseDB:   getEnv("CLICKHOUSE_DB", "reports"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
