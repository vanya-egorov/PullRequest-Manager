package config

import (
	"os"
)

type Config struct {
	HTTPAddr    string
	DBURL       string
	AdminToken  string
	UserToken   string
	Migrate     bool
	Environment string
}

func Load() Config {
	cfg := Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DBURL:       getEnv("DB_URL", "postgres://postgres:postgres@db:5432/pr_review?sslmode=disable"),
		AdminToken:  getEnv("ADMIN_TOKEN", "admin-secret"),
		UserToken:   getEnv("USER_TOKEN", "user-secret"),
		Migrate:     getEnv("RUN_MIGRATIONS", "true") == "true",
		Environment: getEnv("ENVIRONMENT", "local"),
	}
	return cfg
}

func getEnv(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}
