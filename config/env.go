package config

import (
	"os"

	"github.com/joho/godotenv"
)

// EnvKey defines typed environment variable keys to prevent magic string typo bugs.
type EnvKey string

const (
	PORT                   EnvKey = "PORT"
	WORKER_ID              EnvKey = "WORKER_ID"
	API_BASE_URL           EnvKey = "API_BASE_URL"
	API_AUTH_TOKEN         EnvKey = "API_AUTH_TOKEN"
	API_KEY_HEADER         EnvKey = "API_KEY_HEADER"
	API_KEY                EnvKey = "API_KEY"
	MQTT_BROKER            EnvKey = "MQTT_BROKER"
	MQTT_CLIENT_ID         EnvKey = "MQTT_CLIENT_ID"
	MQTT_USERNAME          EnvKey = "MQTT_USERNAME"
	MQTT_PASSWORD          EnvKey = "MQTT_PASSWORD"
	MQTT_CAMERA_TOPIC      EnvKey = "MQTT_CAMERA_TOPIC"
	RETRY_INTERVAL_SECONDS EnvKey = "RETRY_INTERVAL_SECONDS"
	FFMPEG_PATH            EnvKey = "FFMPEG_PATH"
)

// LoadEnv loads environment variables from .env file if available.
func LoadEnv() error {
	return godotenv.Load()
}

// GetValue returns the string value of the environment variable.
func (e EnvKey) GetValue() string {
	return os.Getenv(string(e))
}

// GetValueOrDefault returns the environment variable value or fallback if not set.
func (e EnvKey) GetValueOrDefault(defaultVal string) string {
	val := os.Getenv(string(e))
	if val == "" {
		return defaultVal
	}
	return val
}
