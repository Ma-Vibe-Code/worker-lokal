package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/Ma-Vibe-Code/worker-lokal/internal/client"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/models"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/streamer"
)

// Subscriber manages MQTT subscription to worker event topics and triggers stream actions.
type Subscriber struct {
	cfg           *config.Config
	apiClient     *client.APIClient
	streamManager *streamer.StreamManager
	client        mqtt.Client
	topic         string
}

// NewSubscriber creates and configures an MQTT subscriber instance.
func NewSubscriber(
	cfg *config.Config,
	apiClient *client.APIClient,
	streamManager *streamer.StreamManager,
) *Subscriber {
	topic := fmt.Sprintf("workers/%s/events", cfg.WorkerID)
	return &Subscriber{
		cfg:           cfg,
		apiClient:     apiClient,
		streamManager: streamManager,
		topic:         topic,
	}
}

// Connect initializes the MQTT client and establishes the connection.
func (s *Subscriber) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(s.cfg.MQTTBroker)
	opts.SetClientID(s.cfg.MQTTClientID)

	if s.cfg.MQTTUsername != "" {
		opts.SetUsername(s.cfg.MQTTUsername)
		opts.SetPassword(s.cfg.MQTTPassword)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("[MQTT] Connected to broker %s successfully", s.cfg.MQTTBroker)
		log.Printf("[MQTT] Subscribing to specific topic '%s' (QoS: 1)...", s.topic)

		token := c.Subscribe(s.topic, 1, s.handleMessage)
		if token.Wait() && token.Error() != nil {
			log.Printf("[MQTT] Failed to subscribe to topic '%s': %v", s.topic, token.Error())
		} else {
			log.Printf("[MQTT] Subscribed to specific topic '%s' successfully", s.topic)
		}

		broadcastTopic := "workers/events"
		log.Printf("[MQTT] Subscribing to broadcast topic '%s' (QoS: 1)...", broadcastTopic)
		bToken := c.Subscribe(broadcastTopic, 1, s.handleMessage)
		if bToken.Wait() && bToken.Error() != nil {
			log.Printf("[MQTT] Failed to subscribe to broadcast topic '%s': %v", broadcastTopic, bToken.Error())
		} else {
			log.Printf("[MQTT] Subscribed to broadcast topic '%s' successfully", broadcastTopic)
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v. Auto-reconnecting...", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Printf("[MQTT] Reconnecting to broker %s...", s.cfg.MQTTBroker)
	})

	client := mqtt.NewClient(opts)
	log.Printf("[MQTT] Connecting to broker %s (ClientID: %s)...", s.cfg.MQTTBroker, s.cfg.MQTTClientID)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT connection error: %w", token.Error())
	}

	s.client = client
	return nil
}

// Disconnect gracefully terminates the MQTT client connection.
func (s *Subscriber) Disconnect() {
	if s.client != nil && s.client.IsConnected() {
		log.Println("[MQTT] Disconnecting from broker...")
		s.client.Unsubscribe(s.topic, "workers/events")
		s.client.Disconnect(250)
		log.Println("[MQTT] Disconnected cleanly.")
	}
}

// handleMessage processes incoming JSON MQTT event payloads.
func (s *Subscriber) handleMessage(_ mqtt.Client, msg mqtt.Message) {
	log.Printf("[MQTT] Event received on topic [%s]: %s", msg.Topic(), string(msg.Payload()))

	var event models.MQTTEventPayload
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		log.Printf("[MQTT] Error decoding event payload: %v", err)
		return
	}

	switch event.Action {
	case models.ActionSyncAll:
		log.Println("[MQTT] Action received: SYNC_ALL -> Re-fetching cameras from REST API...")
		go s.handleSyncAll()

	case models.ActionUpsertCamera:
		if event.Camera == nil {
			log.Println("[MQTT] Action UPSERT_CAMERA received but 'camera' payload is missing")
			return
		}
		log.Printf("[MQTT] Action received: UPSERT_CAMERA -> Upserting camera %s (%s)", event.Camera.ID, event.Camera.Name)
		s.streamManager.UpsertCamera(*event.Camera)

	case models.ActionRemoveCamera:
		if event.CameraID == "" {
			log.Println("[MQTT] Action REMOVE_CAMERA received but 'camera_id' is missing")
			return
		}
		log.Printf("[MQTT] Action received: REMOVE_CAMERA -> Removing camera %s", event.CameraID)
		s.streamManager.RemoveCamera(event.CameraID)

	default:
		log.Printf("[MQTT] Unknown action '%s' received, ignoring", event.Action)
	}
}

func (s *Subscriber) handleSyncAll() {
	cameras, err := s.apiClient.FetchCameras()
	if err != nil {
		log.Printf("[MQTT] SYNC_ALL failed to fetch cameras from API: %v", err)
		return
	}
	s.streamManager.ReconcileCameras(cameras)
}
