package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Ma-Vibe-Code/worker-lokal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/handler/message_broker"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/model"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/service"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/utils"
)

// HealthHandler provides telemetry and operational health check endpoints.
type HealthHandler struct {
	startTime     time.Time
	streamManager *service.StreamManagerService
}

// NewHealthHandler creates an instance of HealthHandler.
func NewHealthHandler(streamManager *service.StreamManagerService) *HealthHandler {
	return &HealthHandler{
		startTime:     time.Now(),
		streamManager: streamManager,
	}
}

// CheckHealth returns the operational health status and active stream count.
// @Summary Check worker health status
// @Description Returns operational metrics, active stream count, and MQTT connectivity status
// @Tags Health
// @Produce json
// @Success 200 {object} model.HealthStatusResponse "Worker is healthy"
// @Failure 500 {object} model.ResponseError[string] "Internal Server Error"
// @Router /api/v1/health [get]
func (h *HealthHandler) CheckHealth(c *fiber.Ctx) error {
	mqttConnected := false
	if message_broker.DefaultMQTTBroker != nil {
		mqttConnected = message_broker.DefaultMQTTBroker.IsConnected()
	}

	status := model.HealthStatus{
		Status:        "OK",
		WorkerID:      config.WORKER_ID.GetValue(),
		ActiveStreams: h.streamManager.ActiveStreamCount(),
		MQTTConnected: mqttConnected,
		Uptime:        time.Since(h.startTime).Round(time.Second).String(),
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Worker service is running normally", status, nil)
}
