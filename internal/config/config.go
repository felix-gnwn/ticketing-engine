package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                string
	GRPCPort               string
	JWTSecret              string
	DBConnString           string
	RedisHost              string
	RedisPort              string
	RedisPassword          string
	MinIOEndpoint          string
	MinIORegion            string
	MinIOAccessKey         string
	MinIOSecretKey         string
	MinIOBucket            string
	KafkaBrokers           []string
	KafkaTopicOrderCreated string
	WorkerCount            int
	WorkerBufferSize       int
	KafkaConsumerGroup     string
	KafkaMinBytes          int
	KafkaMaxBytes          int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	brokersRaw := getEnv("KAFKA_BROKERS", "localhost:19092")

	cfg := &Config{
		AppPort:                getEnv("PORT", "8080"),
		GRPCPort:               getEnv("GRPC_PORT", "50051"),
		JWTSecret:              getEnv("JWT_SECRET", ""),
		DBConnString:           getEnv("DB_CONNECTION_STRING", ""),
		RedisHost:              getEnv("REDIS_HOST", "localhost"),
		RedisPort:              getEnv("REDIS_PORT", "6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		MinIOEndpoint:          getEnv("MINIO_ENDPOINT", ""),
		MinIORegion:            getEnv("MINIO_REGION", "us-east-1"),
		MinIOAccessKey:         getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:         getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:            getEnv("MINIO_BUCKET_NAME", ""),
		KafkaBrokers:           strings.Split(brokersRaw, ","),
		KafkaTopicOrderCreated: getEnv("KAFKA_TOPIC_ORDER_CREATED", "order-created-events"),
		WorkerCount:            getEnvAsInt("WORKER_COUNT", 3),
		WorkerBufferSize:       getEnvAsInt("WORKER_BUFFER_SIZE", 100),
		KafkaConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP_ID", "ticketing-order-worker-group"),
		KafkaMinBytes:          getEnvAsInt("KAFKA_MIN_BYTES", 10),           // Default 10B
		KafkaMaxBytes:          getEnvAsInt("KAFKA_MAX_BYTES", 10*1024*1024), // Default 10MB
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

func getEnvAsInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists && valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}

	return fallback
}
