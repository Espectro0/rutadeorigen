package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Action string

const (
	ActionCreateLot     Action = "lot:create"
	ActionReadLot       Action = "lot:read"
	ActionCreateFarm    Action = "farm:create"
	ActionReadFarm      Action = "farm:read"
	ActionRegisterStage Action = "lot:register_stage"
	ActionDeleteFarm    Action = "farm:delete"
	ActionTransferLot   Action = "lot:transfer"
	ActionVoidLot       Action = "lot:void"
	ActionReadActor     Action = "actor:read"
	ActionGenerateQR    Action = "qr:generate"
)

type Subject struct {
	ActorID uuid.UUID
	Role    actor.Type
}

type ResourceType string

const (
	ResourceFarm  ResourceType = "farm"
	ResourceLot   ResourceType = "lot"
	ResourceActor ResourceType = "actor"
)

type Resource struct {
	Type        ResourceType
	ID          uuid.UUID
	OwnerID     uuid.UUID
	CustodianID uuid.UUID
	Stage       lot.Stage
}

type Authorizer interface {
	IsAllowed(ctx context.Context, subject Subject, action Action, resource Resource) (bool, error)
}
