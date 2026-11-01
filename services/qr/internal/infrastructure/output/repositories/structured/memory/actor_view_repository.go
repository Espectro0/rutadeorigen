package memory

import (
	"context"
	"sync"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
)

var _ ports.ActorViewRepository = (*ActorViewRepository)(nil)

type ActorViewRepository struct {
	mu     sync.RWMutex
	byUser map[string]*actorview.ActorView
}

func NewActorViewRepository() *ActorViewRepository {
	return &ActorViewRepository{byUser: make(map[string]*actorview.ActorView)}
}

func (r *ActorViewRepository) FindByUserID(_ context.Context, userID string) (*actorview.ActorView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	view, ok := r.byUser[userID]
	if !ok {
		return actorview.Default(), nil
	}
	return view, nil
}

func (r *ActorViewRepository) Save(_ context.Context, view *actorview.ActorView) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byUser[view.UserID()] = view
	return nil
}
