package enum

// MQTTEventAction defines recognized action types for real-time worker synchronization events.
type MQTTEventAction string

const (
	ActionSyncAll      MQTTEventAction = "SYNC_ALL"
	ActionUpsertCamera MQTTEventAction = "UPSERT_CAMERA"
	ActionRemoveCamera MQTTEventAction = "REMOVE_CAMERA"
)
