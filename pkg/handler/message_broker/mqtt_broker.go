package message_broker

import (
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2/log"

	"github.com/Ma-Vibe-Code/worker-lokal/config"
)

// MQTTBroker wraps the underlying Paho MQTT client providing standard Consume/Publish helpers.
type topicSubscription struct {
	topic   string
	qos     byte
	handler mqtt.MessageHandler
}

type MQTTBroker struct {
	client        mqtt.Client
	mu            sync.RWMutex
	subscriptions map[string]topicSubscription
}

// DefaultMQTTBroker holds the singleton active broker instance.
var DefaultMQTTBroker *MQTTBroker

// InitMQTTBroker initializes and connects the MQTT client instance using typed config values.
func InitMQTTBroker() (*MQTTBroker, error) {
	brokerURI := config.MQTT_BROKER.GetValue()
	workerID := config.WORKER_ID.GetValue()
	clientID := config.MQTT_CLIENT_ID.GetValueOrDefault(workerID)
	username := config.MQTT_USERNAME.GetValue()
	password := config.MQTT_PASSWORD.GetValue()

	if brokerURI == "" {
		return nil, fmt.Errorf("MQTT_BROKER is not configured")
	}

	broker := &MQTTBroker{
		subscriptions: make(map[string]topicSubscription),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetClientID(clientID)

	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Infof("[MQTT] Connected to broker %s successfully (ClientID: %s)", brokerURI, clientID)
		broker.resubscribeAll()
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Warnf("[MQTT] Connection lost: %v. Reconnecting automatically...", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Infof("[MQTT] Reconnecting to broker %s...", brokerURI)
	})

	client := mqtt.NewClient(opts)
	broker.client = client
	log.Infof("[MQTT] Connecting to broker %s (ClientID: %s)...", brokerURI, clientID)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("MQTT connection error: %w", token.Error())
	}

	DefaultMQTTBroker = broker
	return broker, nil
}

// resubscribeAll re-registers all tracked subscriptions on the active MQTT client.
func (b *MQTTBroker) resubscribeAll() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.client == nil || !b.client.IsConnected() {
		return
	}

	for topic, sub := range b.subscriptions {
		log.Infof("[MQTT] (Re)subscribing to topic '%s' (QoS: %d)...", topic, sub.qos)
		token := b.client.Subscribe(sub.topic, sub.qos, sub.handler)
		if token.Wait() && token.Error() != nil {
			log.Errorf("[MQTT] Failed to (re)subscribe to topic '%s': %v", topic, token.Error())
		} else {
			log.Infof("[MQTT] Successfully subscribed to topic '%s'", topic)
		}
	}
}

// ConsumeMQTTTopic subscribes to a topic and routes incoming messages to the provided handler function.
func (b *MQTTBroker) ConsumeMQTTTopic(topic string, qos byte, handler mqtt.MessageHandler) error {
	b.mu.Lock()
	b.subscriptions[topic] = topicSubscription{
		topic:   topic,
		qos:     qos,
		handler: handler,
	}
	b.mu.Unlock()

	if b.client == nil || !b.client.IsConnected() {
		log.Warnf("[MQTT] Client not connected yet. Topic '%s' registered for auto-subscription on connect.", topic)
		return nil
	}

	log.Infof("[MQTT] Subscribing to topic '%s' (QoS: %d)...", topic, qos)
	token := b.client.Subscribe(topic, qos, handler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic '%s': %w", topic, token.Error())
	}

	log.Infof("[MQTT] Subscribed to topic '%s' successfully", topic)
	return nil
}

// PublishToMqtt publishes a payload to the given topic.
func (b *MQTTBroker) PublishToMqtt(topic string, qos byte, retained bool, payload interface{}) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	token := b.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic '%s': %w", topic, token.Error())
	}
	return nil
}

// IsConnected returns whether the MQTT client is currently connected.
func (b *MQTTBroker) IsConnected() bool {
	return b.client != nil && b.client.IsConnected()
}

// Disconnect gracefully disconnects the MQTT broker client.
func (b *MQTTBroker) Disconnect() {
	if b.client != nil && b.client.IsConnected() {
		log.Info("[MQTT] Disconnecting from MQTT broker...")
		b.client.Disconnect(250)
		log.Info("[MQTT] MQTT client disconnected cleanly.")
	}
}

