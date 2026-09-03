package consumer

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2/log"

	"github.com/Ma-Vibe-Code/worker-lokal/pkg/dto"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/enum"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/service"
)

// CameraConsumer handles real-time MQTT synchronization events dispatched by the backend.
type CameraConsumer struct {
	cameraClient  *service.CameraClientService
	streamManager *service.StreamManagerService
}

// NewCameraConsumer creates an instance of CameraConsumer with injected dependencies.
func NewCameraConsumer(
	cameraClient *service.CameraClientService,
	streamManager *service.StreamManagerService,
) *CameraConsumer {
	return &CameraConsumer{
		cameraClient:  cameraClient,
		streamManager: streamManager,
	}
}

// HandleEventMessage unmarshals incoming MQTT event payloads and triggers corresponding service methods.
func (c *CameraConsumer) HandleEventMessage(_ mqtt.Client, msg mqtt.Message) {
	log.Infof("[CONSUMER][MQTT] Event received on topic [%s]: %s", msg.Topic(), string(msg.Payload()))

	var event dto.MQTTEventPayloadDTO
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		log.Errorf("[CONSUMER][MQTT] Failed to decode event JSON: %v", err)
		return
	}

	switch event.Action {
	case enum.ActionSyncAll:
		log.Info("[CONSUMER][MQTT] Action SYNC_ALL -> Re-fetching camera list from backend API...")
		go c.handleSyncAll()

	case enum.ActionUpsertCamera:
		if event.Camera == nil {
			log.Warn("[CONSUMER][MQTT] Action UPSERT_CAMERA received without camera payload")
			return
		}
		log.Infof("[CONSUMER][MQTT] Action UPSERT_CAMERA -> Upserting camera %s (%s)", event.Camera.ID, event.Camera.Name)
		c.streamManager.UpsertCamera(*event.Camera)

	case enum.ActionRemoveCamera:
		if event.CameraID == "" {
			log.Warn("[CONSUMER][MQTT] Action REMOVE_CAMERA received without camera_id")
			return
		}
		log.Infof("[CONSUMER][MQTT] Action REMOVE_CAMERA -> Removing camera %s", event.CameraID)
		c.streamManager.RemoveCamera(event.CameraID)

	default:
		log.Warnf("[CONSUMER][MQTT] Unknown action '%s' received, ignoring", event.Action)
	}
}

func (c *CameraConsumer) handleSyncAll() {
	cameras, err := c.cameraClient.FetchCameras()
	if err != nil {
		log.Errorf("[CONSUMER][MQTT] SYNC_ALL camera fetch failed: %v", err)
		return
	}
	c.streamManager.ReconcileCameras(cameras)
}
