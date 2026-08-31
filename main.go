package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/Ma-Vibe-Code/worker-lokal/internal/client"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/mqtt"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/streamer"
)

func main() {
	log.Println("=================================================================")
	log.Println("  Lightweight Local RTSP-to-MediaMTX Stream Relay Worker")
	log.Println("  Way Kambas Wildlife Surveillance System (Golang)")
	log.Println("=================================================================")

	// 1. Load configuration from .env / OS environment
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Configuration load error: %v", err)
	}

	log.Printf("[MAIN] Worker ID     : %s", cfg.WorkerID)
	log.Printf("[MAIN] API Base URL  : %s", cfg.APIBaseURL)
	log.Printf("[MAIN] API Key Header: %s", cfg.APIKeyHeader)
	if cfg.APIKey != "" {
		log.Printf("[MAIN] API Key       : [CONFIGURED]")
	} else {
		log.Printf("[MAIN] API Key       : [NOT SET]")
	}
	log.Printf("[MAIN] MQTT Broker   : %s", cfg.MQTTBroker)
	log.Printf("[MAIN] Client ID     : %s", cfg.MQTTClientID)
	log.Printf("[MAIN] Retry Interval: %d second(s)", cfg.RetryIntervalSeconds)
	log.Printf("[MAIN] FFmpeg Path   : %s", cfg.FFmpegPath)

	// Verify FFmpeg binary existence
	if _, err := exec.LookPath(cfg.FFmpegPath); err != nil {
		log.Printf("[WARNING] FFmpeg binary '%s' not found in system PATH. Please ensure FFmpeg is installed.", cfg.FFmpegPath)
	} else {
		log.Printf("[MAIN] FFmpeg binary verified: %s", cfg.FFmpegPath)
	}

	// 2. Initialize Stream Manager
	streamMgr := streamer.NewStreamManager(cfg)

	// 3. Initialize REST API Client
	apiClient := client.NewAPIClient(cfg)

	// 4. Initial Bootstrap: Fetch camera list from REST API
	log.Println("[MAIN] Starting initial camera bootstrap via REST API...")
	cameras, err := apiClient.FetchCameras()
	if err != nil {
		log.Printf("[WARNING] Initial camera fetch from REST API failed: %v. Will continue and rely on MQTT events or retry...", err)
	} else {
		log.Printf("[MAIN] Bootstrapping %d camera stream(s)...", len(cameras))
		streamMgr.ReconcileCameras(cameras)
	}

	// 5. Initialize & Connect MQTT Subscriber for Real-Time Events
	subscriber := mqtt.NewSubscriber(cfg, apiClient, streamMgr)
	if err := subscriber.Connect(); err != nil {
		log.Printf("[WARNING] Failed to connect to MQTT broker on startup: %v. (Subscriber will attempt background reconnection)", err)
	}

	log.Println("[MAIN] Worker is now running and awaiting MQTT events. Press Ctrl+C to terminate.")

	// 6. Graceful Shutdown listener for OS interrupt / terminate signals
	shutdownSig := make(chan os.Signal, 1)
	signal.Notify(shutdownSig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Block until a signal is received
	sig := <-shutdownSig
	log.Printf("[MAIN] Received shutdown signal (%s). Initiating graceful shutdown...", sig.String())

	// Step A: Disconnect MQTT client
	subscriber.Disconnect()

	// Step B: Terminate all active FFmpeg child processes
	streamMgr.StopAll()

	log.Println("=================================================================")
	log.Println("  Worker shutdown completed successfully. Goodbye!")
	log.Println("=================================================================")
}
