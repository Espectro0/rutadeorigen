package memory

import (
	"context"
	"sync"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/redirect"
)

var _ ports.RedirectRepository = (*RedirectRepository)(nil)

type RedirectRepository struct {
	mu    sync.RWMutex
	rules []*redirect.Rule
}

func NewRedirectRepository(rules ...*redirect.Rule) *RedirectRepository {
	return &RedirectRepository{rules: append([]*redirect.Rule(nil), rules...)}
}

func (r *RedirectRepository) ListActive(_ context.Context) ([]*redirect.Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := make([]*redirect.Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Active() {
			active = append(active, rule)
		}
	}
	return active, nil
}
