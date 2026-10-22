package ports

import (
	"context"

	"github.com/google/uuid"
)

type Action string

const (
	ActionReadQRImage Action = "qr:image:read"
)

type Subject struct {
	ActorID uuid.UUID
}

type ResourceType string

const (
	ResourceLot ResourceType = "lot"
)

type Resource struct {
	Type        ResourceType
	ID          uuid.UUID
	CustodianID uuid.UUID
}

type Authorizer interface {
	IsAllowed(ctx context.Context, subject Subject, action Action, resource Resource) (bool, error)
}
