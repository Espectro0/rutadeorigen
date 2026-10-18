package listlots

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

var _ UseCase = (*ListLots)(nil)

type ListLots struct {
	actors ports.ActorRepository
	lots   ports.LotRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, lots ports.LotRepository, auth ports.Authorizer) *ListLots {
	return &ListLots{actors: actors, lots: lots, auth: auth}
}

func (uc *ListLots) Execute(ctx context.Context, q Query) (lot.Page, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return lot.Page{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadLot,
		ports.Resource{Type: ports.ResourceLot})
	if err != nil {
		return lot.Page{}, exception.Wrap(err, "evaluate list lots policy")
	}
	if !allowed {
		return lot.Page{}, exception.New(exception.CodeForbidden)
	}

	limit := clampLimit(q.Limit)
	rows, err := uc.lots.List(ctx, q.Filter, q.After, limit+1)
	if err != nil {
		return lot.Page{}, exception.Wrap(err, "list lots")
	}
	return paginate(rows, limit), nil
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultLimit
	case limit > maxLimit:
		return maxLimit
	default:
		return limit
	}
}

func paginate(rows []*lot.Lot, limit int) lot.Page {
	if len(rows) <= limit {
		return lot.Page{Lots: rows}
	}
	page := rows[:limit]
	return lot.Page{Lots: page, Next: lot.CursorOf(page[limit-1])}
}
