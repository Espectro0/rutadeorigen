package registeractor

import "github.com/google/uuid"

type ActorRegisteredPayload struct {
	ActorID uuid.UUID `json:"actor_id"`
	UserID  string    `json:"user_id"`
	Type    string    `json:"type"`
}
