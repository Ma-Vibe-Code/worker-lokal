# [Refactor] Align `worker-lokal` with PT. LSKK Worker Development Standard Stack (v1.0)

## 📌 Background & Objective
Refactoring `worker-lokal` (Local RTSP-to-MediaMTX Stream Relay Worker) to comply with the official **PT. LSKK Worker Development Standard Stack v1.0** (dated 24 August 2026).

Currently, `worker-lokal` operates as a standalone daemon using Go standard library (`log.Println`), root-level `main.go`, and unstructured package layout. This issue outlines the architectural transition to the standardized layered structure, typed configuration, Fiber v2 logging & health check server, and standard message broker handling.

---

## 🏗️ Target Architecture & Folder Layout

Refactoring the project directory from `internal/` into the standardized PT. LSKK layered stack:

```text
worker-lokal/
├── cmd/
│   └── worker/
│       └── main.go                     # Entrypoint, dependency wiring, Fiber server, graceful shutdown
├── config/
│   ├── env.go                          # Typed EnvKey constants & loader (godotenv)
│   └── mqtt.go                         # MQTT Client Connector (per-instance)
├── pkg/
│   ├── dto/                            # DTO definitions (Camera API payload, MQTT event payload)
│   ├── enum/                           # Typed string constants (StreamStatus, EventType, etc.)
│   ├── model/                          # Domain models & standard ResponseEntity[T] / ResponseError[T]
│   ├── handler/
│   │   ├── http/                       # Fiber HTTP handlers (Health check, Stream Status) + Swaggo annotations
│   │   ├── consumer/                   # MQTT message consumers (Unmarshal -> call Service)
│   │   ├── message_broker/             # Reusable MQTT broker helper wrapper
│   │   └── err/                        # Centralized Fiber ErrorHandler
│   ├── router/                         # HTTP routes & consumer registration
│   ├── service/                        # Business logic (StreamManager, CameraSyncService, APIClient)
│   └── utils/                          # Standard SuccessResponse / ErrorResponse helpers
├── docs/                               # Auto-generated Swagger documentation (swag init)
├── .env.example                        # Template environment variables (safe defaults, no real secrets)
├── go.mod                              # Go 1.25+ dependencies (Fiber v2, Paho MQTT, Swaggo, godotenv)
└── go.sum
```

---

## 📋 Task Checklist & Implementation Breakdown

### Phase 1: Go Dependencies & Project Setup
- [ ] Upgrade / align `go.mod` module to target Go 1.25+.
- [ ] Install required standard dependencies:
  - `github.com/gofiber/fiber/v2` (HTTP framework & logger)
  - `github.com/gofiber/swagger` + `github.com/swaggo/swag` (API docs)
  - `github.com/eclipse/paho.mqtt.golang` (MQTT client)
  - `github.com/joho/godotenv` (Environment loader)

### Phase 2: Configuration & Environment Management (`config/`)
- [ ] Create `config/env.go` with typed `type EnvKey string` constants:
  - `PORT`, `WORKER_ID`, `API_BASE_URL`, `API_AUTH_TOKEN`, `API_KEY_HEADER`, `API_KEY`
  - `MQTT_BROKER`, `MQTT_CLIENT_ID`, `MQTT_USERNAME`, `MQTT_PASSWORD`, `MQTT_TOPIC_CAMERA_EVENTS`
  - `RETRY_INTERVAL_SECONDS`, `FFMPEG_PATH`, `MEDIAMTX_RTSP_BASE_URL`
- [ ] Implement `(e EnvKey) GetValue() string` and `(e EnvKey) GetValueOrDefault(defaultVal string) string`.
- [ ] Update `.env.example` with safe placeholder defaults.

### Phase 3: Response Wrapper & Centralized Error Handling (`pkg/model/`, `pkg/utils/`, `pkg/handler/err/`)
- [ ] Implement standard response wrappers:
  - `MetaPagination`, `ResponseEntity[T]`, `ResponseError[T]` in `pkg/model/response_struct.go`.
- [ ] Implement `SuccessResponse[T]()` and `ErrorResponse[T]()` in `pkg/utils/response.go`.
- [ ] Implement centralized `ErrorHandler(c *fiber.Ctx, err error) error` in `pkg/handler/err/error_handler.go`.

### Phase 4: Standard Logger Migration (`fiber/v2/log`)
- [ ] Replace all standard Go `log.Println` / `log.Printf` with `github.com/gofiber/fiber/v2/log`:
  - `log.Infof()`, `log.Warnf()`, `log.Errorf()`, `log.Debugf()`.
- [ ] Prohibit `fmt.Println` and standard `log` package in application logic.

### Phase 5: Service Layer Refactoring (`pkg/service/`)
- [ ] Refactor REST API Client into `pkg/service/camera_client_service.go`.
- [ ] Refactor FFmpeg Process Manager into `pkg/service/stream_manager_service.go`.
- [ ] Implement constructor pattern: `NewStreamManagerService(cfg *config.Config)`, `NewCameraClientService(cfg *config.Config)`.

### Phase 6: Message Broker & Consumer Layer (`pkg/handler/message_broker/`, `pkg/handler/consumer/`)
- [ ] Create generic MQTT broker wrapper in `pkg/handler/message_broker/mqtt_broker.go` (`ConsumeMQTTTopic`, `PublishToMqtt`).
- [ ] Create dedicated MQTT event consumer in `pkg/handler/consumer/camera_consumer.go`:
  - Responsibility strictly limited to: Unmarshal MQTT payload $\to$ validate DTO $\to$ invoke `StreamManagerService`.
  - Zero business logic inside consumer callbacks.

### Phase 7: HTTP Server, Router & Swagger Docs (`pkg/handler/http/`, `pkg/router/`, `docs/`)
- [ ] Implement `pkg/handler/http/health_handler.go`:
  - `GET /health` (System status, active FFmpeg stream count, MQTT connection status).
  - Include full Swaggo annotations (`@Summary`, `@Tags`, `@Produce`, `@Success`, `@Failure`, `@Router`).
- [ ] Implement Fiber Middlewares: `logger.New()`, `recover.New()`, `cors.New()`, `compress.New()`.
- [ ] Generate Swagger docs via `swag init -g cmd/worker/main.go -o docs`.

### Phase 8: Entrypoint & Graceful Shutdown (`cmd/worker/main.go`)
- [ ] Assemble manual Dependency Injection in `cmd/worker/main.go`.
- [ ] Implement `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)`:
  - Stop Fiber HTTP server (`app.ShutdownWithTimeout`).
  - Terminate all active FFmpeg relay processes cleanly (`streamManager.StopAll()`).
  - Disconnect MQTT broker client cleanly.

---

## 🎯 Acceptance Criteria
1. `worker-lokal` builds cleanly with `go build ./cmd/worker/main.go`.
2. Folder layout matches PT. LSKK Standard Stack v1.0 specifications.
3. Fiber v2 HTTP server responds with `ResponseEntity` on `GET /health` and Swagger UI is accessible at `/swagger/index.html`.
4. Camera stream reconciliation via REST API and MQTT real-time synchronization function without stream interruption or memory leaks.
5. Graceful shutdown halts FFmpeg child processes and releases system ports within <2 seconds.
