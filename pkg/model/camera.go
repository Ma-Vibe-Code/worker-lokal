package model

import "time"

// Camera represents the configuration and status of a single CCTV camera stream entity.
type Camera struct {
	ID        string    `json:"id" bson:"id"`
	Name      string    `json:"name" bson:"name"`
	IsActive  bool      `json:"is_active" bson:"is_active"`
	SourceURL string    `json:"source_url" bson:"source_url"`
	TargetURL string    `json:"target_url" bson:"target_url"`
	CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

// HealthStatus represents the payload returned by the health check endpoint.
type HealthStatus struct {
	Status        string `json:"status"`
	WorkerID      string `json:"worker_id"`
	ActiveStreams int    `json:"active_streams"`
	MQTTConnected bool   `json:"mqtt_connected"`
	Uptime        string `json:"uptime"`
}

// HealthStatusResponse provides a concrete envelope for Swagger documentation.
type HealthStatusResponse struct {
	ResponseEntity[HealthStatus]
}
