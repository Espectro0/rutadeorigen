package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

func TestActorRepositoryListByTypeFiltersByTypeInTheCurrentTransaction(t *testing.T) {
	first := repoActor()
	second := actor.Restore(uuid.New(), "user_02", actor.TypeProducer, "Finca El Roble",
		"user_02@example.com", actor.StatusActive, first.CreatedAt())
	tx := &actorTx{rows: []entity.Actor{actorRow(first), actorRow(second)}}

	got, err := NewActorRepository(nil).ListByType(actorCtx(tx), actor.TypeProducer)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.querySQL != selectActorsByType || len(tx.queryArgs) != 1 || tx.queryArgs[0] != "producer" {
		t.Errorf("unexpected query %q with args %v", tx.querySQL, tx.queryArgs)
	}
	if len(got) != 2 || got[0].ID() != first.ID() || got[1].ID() != second.ID() {
		t.Errorf("got %v actors, want the two stored ones in order", len(got))
	}
}

func TestActorRepositoryListByTypeReturnsAnEmptyListWhenNothingMatches(t *testing.T) {
	got, err := NewActorRepository(nil).ListByType(actorCtx(&actorTx{}), actor.TypeRoaster)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got %v, want an empty non nil list", got)
	}
}

func TestActorRepositoryListByTypeFailures(t *testing.T) {
	cause := errors.New("connection reset")
	tests := []struct {
		name string
		tx   *actorTx
	}{
		{"query", &actorTx{queryErr: cause}},
		{"scan", &actorTx{rows: []entity.Actor{actorRow(repoActor())}, scanErr: cause}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewActorRepository(nil).ListByType(actorCtx(tt.tx), actor.TypeProducer)

			if !exception.HasCode(err, exception.CodeInternal) {
				t.Errorf("got %v, want code %q", err, exception.CodeInternal)
			}
		})
	}
}

func TestActorListQueryFiltersByTypeAndOrdersByName(t *testing.T) {
	for _, column := range actorColumns {
		if !strings.Contains(selectActorsByType, column) {
			t.Errorf("query must select %q", column)
		}
	}
	if !strings.Contains(selectActorsByType, "WHERE type = $1") || !strings.Contains(selectActorsByType, "ORDER BY name ASC, id ASC") {
		t.Error("the query must filter by type and order by name")
	}
}
