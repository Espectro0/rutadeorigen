package response

import "github.com/google/uuid"

type ResolvedActor struct {
	ActorID uuid.UUID
	Role    string
}
