package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"

	"github.com/Ma-Vibe-Code/worker-lokal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/handler/consumer"
	apperr "github.com/Ma-Vibe-Code/worker-lokal/pkg/handler/err"
	apphttp "github.com/Ma-Vibe-Code/worker-lokal/pkg/handler/http"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/handler/message_broker"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/router"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/service"
)

// @title Worker Lokal RTSP Relay API
// @version 1.0
// @description Local RTSP-to-MediaMTX Stream Relay Worker for Way Kambas Wildlife Surveillance System
// @BasePath /
func main() {
	log.Info("=================================================================")
	log.Info("  Lightweight Local RTSP-to-MediaMTX Stream Relay Worker")
	log.Info("  Way Kambas Wildlife Surveillance System (PT. LSKK Standard)")
	log.Info("=================================================================")

	// Phase 1: Load Environment Variables
	if err := config.LoadEnv(); err != nil {
		log.Warn("[MAIN] No .env file found, using system environment variables")
	}

	workerID := config.WORKER_ID.GetValue()
	if workerID == "" {
		log.Fatal("[FATAL] WORKER_ID must be configured in environment or .env")
	}

	httpPort := config.PORT.GetValueOrDefault("3000")
	ffmpegPath := config.FFMPEG_PATH.GetValueOrDefault("ffmpeg")

	log.Infof("[MAIN] Worker ID     : %s", workerID)
	log.Infof("[MAIN] HTTP Port     : %s", httpPort)
	log.Infof("[MAIN] API Base URL  : %s", config.API_BASE_URL.GetValue())
	log.Infof("[MAIN] MQTT Broker   : %s", config.MQTT_BROKER.GetValue())
	log.Infof("[MAIN] FFmpeg Path   : %s", ffmpegPath)

	if _, err := exec.LookPath(ffmpegPath); err != nil {
		log.Warnf("[WARNING] FFmpeg binary '%s' not found in system PATH. Ensure FFmpeg is installed.", ffmpegPath)
	} else {
		log.Infof("[MAIN] FFmpeg binary verified: %s", ffmpegPath)
	}

	// Phase 2: Initialize Services (Manual Dependency Injection)
	cameraClientService := service.NewCameraClientService()
	streamManagerService := service.NewStreamManagerService()
	healthHandler := apphttp.NewHealthHandler(streamManagerService)
	cameraConsumer := consumer.NewCameraConsumer(cameraClientService, streamManagerService)

	// Phase 3: Initialize Fiber HTTP Server
	app := fiber.New(fiber.Config{
		ErrorHandler: apperr.ErrorHandler,
		AppName:      "Worker Lokal RTSP Relay v1.0",
	})
	router.SetupRoutes(app, healthHandler)

	go func() {
		addr := fmt.Sprintf(":%s", httpPort)
		log.Infof("[MAIN] Starting Fiber HTTP server on %s...", addr)
		if err := app.Listen(addr); err != nil {
			log.Infof("[MAIN] Fiber server listener stopped: %v", err)
		}
	}()

	// Phase 4: Initial Bootstrap - Fetch cameras from backend REST API
	log.Info("[MAIN] Starting initial camera bootstrap via REST API...")
	cameras, err := cameraClientService.FetchCameras()
	if err != nil {
		log.Warnf("[MAIN] Initial camera fetch from REST API failed: %v. Relying on MQTT events...", err)
	} else {
		log.Infof("[MAIN] Bootstrapping %d camera stream(s)...", len(cameras))
		streamManagerService.ReconcileCameras(cameras)
	}

	// Phase 5: Initialize MQTT Broker & Register Consumers
	mqttBroker, err := message_broker.InitMQTTBroker()
	if err != nil {
		log.Warnf("[MAIN] Failed to connect to MQTT broker on startup: %v. Broker will retry in background...", err)
	} else {
		// Specific worker topic
		workerTopic := fmt.Sprintf("workers/%s/events", workerID)
		_ = mqttBroker.ConsumeMQTTTopic(workerTopic, 1, cameraConsumer.HandleEventMessage)

		// Broadcast topic
		broadcastTopic := "workers/events"
		_ = mqttBroker.ConsumeMQTTTopic(broadcastTopic, 1, cameraConsumer.HandleEventMessage)
	}

	log.Info("[MAIN] Worker is now operational. Awaiting MQTT events / HTTP telemetry...")

	// Phase 6: Graceful Shutdown Handling (signal.NotifyContext)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	<-ctx.Done()
	log.Info("[MAIN] Received shutdown signal. Initiating graceful shutdown sequence...")

	// Step A: Disconnect MQTT client
	if message_broker.DefaultMQTTBroker != nil {
		message_broker.DefaultMQTTBroker.Disconnect()
	}

	// Step B: Halt all active FFmpeg relay subprocesses
	streamManagerService.StopAll()

	// Step C: Shutdown Fiber HTTP server with timeout
	if err := app.ShutdownWithTimeout(3 * time.Second); err != nil {
		log.Errorf("[MAIN] Fiber HTTP server shutdown error: %v", err)
	}

	log.Info("=================================================================")
	log.Info("  Worker shutdown completed successfully. Goodbye!")
	log.Info("=================================================================")
}
