package request

type ProjectActorRegistered struct {
	MessageID  string
	EventType  string
	Version    int
	OccurredAt string
	Data       ActorRegisteredData
}

type ActorRegisteredData struct {
	ActorID string `json:"actor_id"`
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
}
