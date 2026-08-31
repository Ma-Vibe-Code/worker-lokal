package models

// Camera represents the configuration and status of a single CCTV camera stream.
type Camera struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	SourceURL string `json:"source_url"`
	TargetURL string `json:"target_url"`
}

// APIResponse represents the standard response payload from the REST API.
type APIResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message,omitempty"`
	Data    []Camera `json:"data"`
}

// MQTTEventAction defines the recognized real-time action types.
type MQTTEventAction string

const (
	ActionSyncAll      MQTTEventAction = "SYNC_ALL"
	ActionUpsertCamera MQTTEventAction = "UPSERT_CAMERA"
	ActionRemoveCamera MQTTEventAction = "REMOVE_CAMERA"
)

// MQTTEventPayload represents the incoming JSON event message on the worker's MQTT topic.
type MQTTEventPayload struct {
	Action   MQTTEventAction `json:"action"`
	Camera   *Camera         `json:"camera,omitempty"`
	CameraID string          `json:"camera_id,omitempty"`
}
