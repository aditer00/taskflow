package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppName      string
	AppEnv       string
	AppSecret    string
	Port         string
	LogLevel     string
	DatabaseURL  string
	RedisAddr    string
	RedisPass    string
	AWSKey       string
	AWSSecret    string
	AWSRegion    string
	AWSBucket    string
	AWSEndpoint  string
	MailHost     string
	MailPort     string
	MailFrom     string
}

func Load() Config {
	dbHost := envOr("DB_HOST", "postgres")
	dbPort := envOr("DB_PORT", "5432")
	dbName := envOr("DB_DATABASE", "taskflow")
	dbUser := envOr("DB_USERNAME", "taskflow")
	dbPass := envOr("DB_PASSWORD", "secret")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	return Config{
		AppName:     envOr("APP_NAME", "TaskFlow"),
		AppEnv:      envOr("APP_ENV", "local"),
		AppSecret:   envOr("APP_SECRET", "dev-secret-change-in-prod"),
		Port:        envOr("PORT", "8080"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		DatabaseURL: dbURL,
		RedisAddr:   envOr("REDIS_ADDR", "redis:6379"),
		RedisPass:   os.Getenv("REDIS_PASSWORD"),
		AWSKey:      envOr("AWS_ACCESS_KEY_ID", "minioadmin"),
		AWSSecret:   envOr("AWS_SECRET_ACCESS_KEY", "minioadmin"),
		AWSRegion:   envOr("AWS_DEFAULT_REGION", "us-east-1"),
		AWSBucket:   envOr("AWS_BUCKET", "taskflow"),
		AWSEndpoint: envOr("AWS_ENDPOINT", "http://minio:9000"),
		MailHost:    envOr("MAIL_HOST", "mailpit"),
		MailPort:    envOr("MAIL_PORT", "1025"),
		MailFrom:    envOr("MAIL_FROM", "noreply@taskflow.local"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
