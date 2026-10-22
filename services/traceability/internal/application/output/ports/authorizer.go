package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
)

type Action string

const (
	ActionCreateLot  Action = "lot:create"
	ActionReadLot    Action = "lot:read"
	ActionCreateFarm Action = "farm:create"
	ActionReadFarm   Action = "farm:read"
)

type Subject struct {
	ActorID uuid.UUID
	Role    actor.Type
}

type ResourceType string

const (
	ResourceFarm ResourceType = "farm"
	ResourceLot  ResourceType = "lot"
)

type Resource struct {
	Type        ResourceType
	ID          uuid.UUID
	OwnerID     uuid.UUID
	CustodianID uuid.UUID
}

type Authorizer interface {
	IsAllowed(ctx context.Context, subject Subject, action Action, resource Resource) (bool, error)
}
