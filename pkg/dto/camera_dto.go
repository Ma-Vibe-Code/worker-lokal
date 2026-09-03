package dto

import (
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/enum"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/model"
)

// APIResponseDTO represents the standard response payload envelope from the central REST API.
type APIResponseDTO struct {
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Data    []model.Camera `json:"data"`
}

// MQTTEventPayloadDTO represents the real-time event message received over MQTT topics.
type MQTTEventPayloadDTO struct {
	Action   enum.MQTTEventAction `json:"action" validate:"required"`
	Camera   *model.Camera        `json:"camera,omitempty"`
	CameraID string               `json:"camera_id,omitempty"`
}
