package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/domain/lotview"
)

type Action string

const (
	ActionReadQRImage Action = "qr:image:read"
	ActionReadQRStats Action = "qr:stats:read"
)

type Subject struct {
	ActorID uuid.UUID
	Role    actorview.Role
}

type ResourceType string

const (
	ResourceLot ResourceType = "lot"
)

type Resource struct {
	Type        ResourceType
	ID          uuid.UUID
	CustodianID uuid.UUID
	Stage       lotview.Stage
}

type Authorizer interface {
	IsAllowed(ctx context.Context, subject Subject, action Action, resource Resource) (bool, error)
}
