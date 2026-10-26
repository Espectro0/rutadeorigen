package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.ActorRepository = (*ActorRepository)(nil)

type ActorRepository struct {
	mu     sync.RWMutex
	actors map[uuid.UUID]*actor.Actor
}

func NewActorRepository(seed ...*actor.Actor) *ActorRepository {
	r := &ActorRepository{actors: make(map[uuid.UUID]*actor.Actor)}
	for _, a := range seed {
		if !a.IsDefault() {
			r.actors[a.ID()] = a
		}
	}
	return r
}

func (r *ActorRepository) FindByID(_ context.Context, id uuid.UUID) (*actor.Actor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.actors[id]
	if !ok {
		return actor.Default(), nil
	}
	return a, nil
}

func (r *ActorRepository) FindByUserID(_ context.Context, userID string) (*actor.Actor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, a := range r.actors {
		if a.UserID() == userID {
			return a, nil
		}
	}
	return actor.Default(), nil
}

func (r *ActorRepository) Save(_ context.Context, a *actor.Actor) error {
	if a.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default actor")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, existing := range r.actors {
		if existing.UserID() == a.UserID() && id != a.ID() {
			return exception.New(exception.CodeActorAlreadyExists)
		}
	}

	r.actors[a.ID()] = a
	return nil
}
