package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	JWTSecret      string
	DBConnString   string
	RedisHost      string
	RedisPort      string
	RedisPassword  string
	MinIOEndpoint  string
	MinIORegion    string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:        getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		DBConnString:   getEnv("DB_CONNECTION_STRING", ""),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", ""),
		MinIORegion:    getEnv("MINIO_REGION", "us-east-1"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:    getEnv("MINIO_BUCKET_NAME", ""),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("mandatory environment variable JWT_SECRET is missing")
	}

	if cfg.DBConnString == "" {
		return nil, fmt.Errorf("mandatory environment variable DB_CONNECTION_STRING is missing")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}

	return fallback
}
