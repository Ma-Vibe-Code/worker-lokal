package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("WORKER_ID", "test_worker_01")
	os.Setenv("API_BASE_URL", "http://localhost:3000/devices/worker")
	os.Setenv("MQTT_BROKER", "tcp://localhost:1883")
	os.Setenv("RETRY_INTERVAL_SECONDS", "10")

	defer func() {
		os.Unsetenv("WORKER_ID")
		os.Unsetenv("API_BASE_URL")
		os.Unsetenv("MQTT_BROKER")
		os.Unsetenv("RETRY_INTERVAL_SECONDS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.WorkerID != "test_worker_01" {
		t.Errorf("Expected WorkerID 'test_worker_01', got '%s'", cfg.WorkerID)
	}

	if cfg.MQTTClientID != "test_worker_01" {
		t.Errorf("Expected default MQTTClientID 'test_worker_01', got '%s'", cfg.MQTTClientID)
	}

	if cfg.RetryIntervalSeconds != 10 {
		t.Errorf("Expected RetryIntervalSeconds 10, got %d", cfg.RetryIntervalSeconds)
	}

	if cfg.FFmpegPath != "ffmpeg" {
		t.Errorf("Expected default FFmpegPath 'ffmpeg', got '%s'", cfg.FFmpegPath)
	}
}
