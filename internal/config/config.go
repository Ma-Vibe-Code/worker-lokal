package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents all application configuration parameters.
type Config struct {
	WorkerID             string
	APIBaseURL           string
	APIAuthToken         string
	APIKeyHeader         string
	APIKey               string
	MQTTBroker           string
	MQTTClientID         string
	MQTTUsername         string
	MQTTPassword         string
	RetryIntervalSeconds int
	FFmpegPath           string
}

// Load reads configuration from .env file or environment variables.
func Load() (*Config, error) {
	// Attempt to load .env file; if not present, proceed with OS env
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] No .env file found, using system environment variables")
	}

	workerID := getEnv("WORKER_ID", "")
	if workerID == "" {
		return nil, fmt.Errorf("WORKER_ID is required in environment variables or .env")
	}

	apiBaseURL := getEnv("API_BASE_URL", "")
	if apiBaseURL == "" {
		return nil, fmt.Errorf("API_BASE_URL is required in environment variables or .env")
	}

	apiAuthToken := getEnv("API_AUTH_TOKEN", "")
	apiKeyHeader := getEnv("API_KEY_HEADER", "x-api-key")
	apiKey := getEnv("API_KEY", "")

	mqttBroker := getEnv("MQTT_BROKER", "")
	if mqttBroker == "" {
		return nil, fmt.Errorf("MQTT_BROKER is required in environment variables or .env (e.g. tcp://103.xxx.xxx.xxx:1883)")
	}

	mqttClientID := getEnv("MQTT_CLIENT_ID", workerID)
	mqttUsername := getEnv("MQTT_USERNAME", "")
	mqttPassword := getEnv("MQTT_PASSWORD", "")

	retryInterval := 5
	if retryStr := getEnv("RETRY_INTERVAL_SECONDS", "5"); retryStr != "" {
		if val, err := strconv.Atoi(retryStr); err == nil && val > 0 {
			retryInterval = val
		}
	}

	ffmpegPath := getEnv("FFMPEG_PATH", "ffmpeg")

	return &Config{
		WorkerID:             workerID,
		APIBaseURL:           apiBaseURL,
		APIAuthToken:         apiAuthToken,
		APIKeyHeader:         apiKeyHeader,
		APIKey:               apiKey,
		MQTTBroker:           mqttBroker,
		MQTTClientID:         mqttClientID,
		MQTTUsername:         mqttUsername,
		MQTTPassword:         mqttPassword,
		RetryIntervalSeconds: retryInterval,
		FFmpegPath:           ffmpegPath,
	}, nil
}

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return fallback
}
