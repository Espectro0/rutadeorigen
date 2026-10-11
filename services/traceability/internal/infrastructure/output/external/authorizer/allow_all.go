package authorizer

import (
	"context"

	"rutadeorigen/traceability/internal/application/output/ports"
)

var _ ports.Authorizer = AllowAll{}

type AllowAll struct{}

func NewAllowAll() AllowAll {
	return AllowAll{}
}

func (AllowAll) IsAllowed(_ context.Context, _ ports.Subject, _ ports.Action, _ ports.Resource) (bool, error) {
	return true, nil
}
