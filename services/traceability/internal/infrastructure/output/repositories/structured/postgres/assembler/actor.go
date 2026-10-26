package assembler

import (
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.Actor, *actor.Actor] = Actor{}

type Actor struct{}

func (Actor) ToDomain(e entity.Actor) *actor.Actor {
	return actor.Restore(e.ID, e.UserID, actor.ParseType(e.Type), e.Name, e.Email, actor.ParseStatus(e.Status), e.CreatedAt.UTC())
}

func (Actor) ToEntity(a *actor.Actor) entity.Actor {
	if a.IsDefault() {
		return entity.Actor{}
	}

	return entity.Actor{
		ID:        a.ID(),
		UserID:    a.UserID(),
		Type:      a.Type().String(),
		Name:      a.Name(),
		Email:     a.Email(),
		Status:    a.Status().String(),
		CreatedAt: a.CreatedAt().UTC(),
	}
}
