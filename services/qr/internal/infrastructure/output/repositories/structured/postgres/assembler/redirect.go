package assembler

import (
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.Redirect, *redirect.Rule] = Redirect{}

type Redirect struct{}

func (Redirect) ToDomain(e entity.Redirect) *redirect.Rule {
	return redirect.Restore(redirect.RestoreParams{
		ID:             e.ID,
		SourcePrefix:   e.SourcePrefix,
		TargetTemplate: e.TargetTemplate,
		StatusCode:     e.StatusCode,
		Active:         e.Active,
		CreatedAt:      e.CreatedAt.UTC(),
	})
}

func (Redirect) ToEntity(r *redirect.Rule) entity.Redirect {
	if r.IsDefault() {
		return entity.Redirect{}
	}

	return entity.Redirect{
		ID:             r.ID(),
		SourcePrefix:   r.SourcePrefix(),
		TargetTemplate: r.TargetTemplate(),
		StatusCode:     r.StatusCode(),
		Active:         r.Active(),
		CreatedAt:      r.CreatedAt().UTC(),
	}
}
