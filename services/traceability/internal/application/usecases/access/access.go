package access

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

type DeniedPayload struct {
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

func Require(ctx context.Context, auth ports.Authorizer, publisher ports.EventPublisher, a *actor.Actor,
	action ports.Action, resource ports.Resource) error {
	subject := ports.Subject{ActorID: a.ID(), Role: a.Type()}

	allowed, err := auth.IsAllowed(ctx, subject, action, resource)
	if err != nil {
		return exception.Wrap(err, "evaluate access policy")
	}
	if allowed {
		return nil
	}

	denied := envelope.New(ctx, a, ports.EventAccessDenied, ports.AggregateLot, uuid.Nil, 0, DeniedPayload{
		Action:       string(action),
		ResourceType: string(resource.Type),
		ResourceID:   resource.ID.String(),
	})
	if err := publisher.Publish(ctx, denied); err != nil {
		return exception.Wrap(err, "publish access denied")
	}

	return exception.New(exception.CodeForbidden)
}
