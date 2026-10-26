package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

var actorColumns = []string{"id", "user_id", "type", "name", "email", "status", "created_at"}

type actorRows struct {
	pgx.Rows
	rows    []entity.Actor
	scanErr error
	index   int
}

func (r *actorRows) Close()     {}
func (r *actorRows) Err() error { return nil }

func (r *actorRows) FieldDescriptions() []pgconn.FieldDescription {
	fields := make([]pgconn.FieldDescription, len(actorColumns))
	for i, name := range actorColumns {
		fields[i] = pgconn.FieldDescription{Name: name}
	}
	return fields
}

func (r *actorRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *actorRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}

	e := r.rows[r.index-1]
	values := []any{e.ID, e.UserID, e.Type, e.Name, e.Email, e.Status, e.CreatedAt}
	for i, target := range dest {
		switch p := target.(type) {
		case *uuid.UUID:
			*p = values[i].(uuid.UUID)
		case *string:
			*p = values[i].(string)
		case *time.Time:
			*p = values[i].(time.Time)
		default:
			return errors.New("unsupported scan target")
		}
	}
	return nil
}

type actorTx struct {
	pgx.Tx
	execErr  error
	queryErr error
	rows     []entity.Actor
	scanErr  error

	execSQL   string
	execArgs  []any
	querySQL  string
	queryArgs []any
}

func (t *actorTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.execSQL, t.execArgs = sql, args
	return pgconn.CommandTag{}, t.execErr
}

func (t *actorTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	t.querySQL, t.queryArgs = sql, args
	if t.queryErr != nil {
		return nil, t.queryErr
	}
	return &actorRows{rows: t.rows, scanErr: t.scanErr}, nil
}

func actorCtx(tx *actorTx) context.Context {
	return context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
}

func repoActor() *actor.Actor {
	return actor.Restore(uuid.New(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS",
		"user_01@example.com", actor.StatusActive, time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC))
}

func actorRow(a *actor.Actor) entity.Actor {
	return entity.Actor{
		ID:        a.ID(),
		UserID:    a.UserID(),
		Type:      a.Type().String(),
		Name:      a.Name(),
		Email:     a.Email(),
		Status:    a.Status().String(),
		CreatedAt: a.CreatedAt(),
	}
}

func saveActor(tx *actorTx, a *actor.Actor) error {
	return NewActorRepository(nil).Save(actorCtx(tx), a)
}

func TestActorRepositoryFinders(t *testing.T) {
	stored := repoActor()
	cause := errors.New("connection reset")

	finders := []struct {
		name  string
		query string
		arg   any
		find  func(ctx context.Context) (*actor.Actor, error)
	}{
		{
			name:  "by id",
			query: selectActorByID,
			arg:   stored.ID(),
			find: func(ctx context.Context) (*actor.Actor, error) {
				return NewActorRepository(nil).FindByID(ctx, stored.ID())
			},
		},
		{
			name:  "by user id",
			query: selectActorByUserID,
			arg:   stored.UserID(),
			find: func(ctx context.Context) (*actor.Actor, error) {
				return NewActorRepository(nil).FindByUserID(ctx, stored.UserID())
			},
		},
	}

	for _, f := range finders {
		t.Run(f.name+" returns the stored actor", func(t *testing.T) {
			tx := &actorTx{rows: []entity.Actor{actorRow(stored)}}

			got, err := f.find(actorCtx(tx))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tx.querySQL != f.query {
				t.Error("unexpected query")
			}
			if len(tx.queryArgs) != 1 || tx.queryArgs[0] != f.arg {
				t.Errorf("query args = %v, want [%v]", tx.queryArgs, f.arg)
			}
			if got.ID() != stored.ID() || got.UserID() != "user_01" || got.Email() != "user_01@example.com" {
				t.Errorf("got %v, want the stored actor", got)
			}
			if got.Type() != actor.TypeProducer || got.Status() != actor.StatusActive || got.Name() != "Finca La Esperanza SAS" {
				t.Errorf("got %v, want type, status and name to be mapped", got)
			}
			if !got.CreatedAt().Equal(stored.CreatedAt()) {
				t.Errorf("created at = %v, want %v", got.CreatedAt(), stored.CreatedAt())
			}
		})

		t.Run(f.name+" without rows returns the default actor", func(t *testing.T) {
			got, err := f.find(actorCtx(&actorTx{}))

			if err != nil || got == nil || !got.IsDefault() {
				t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
			}
		})

		t.Run(f.name+" fails when the query fails", func(t *testing.T) {
			got, err := f.find(actorCtx(&actorTx{queryErr: cause}))

			assertInternalWithCause(t, err, cause)
			if got == nil || !got.IsDefault() {
				t.Errorf("got %v, want actor.Default()", got)
			}
		})

		t.Run(f.name+" fails when the row cannot be scanned", func(t *testing.T) {
			tx := &actorTx{rows: []entity.Actor{actorRow(stored)}, scanErr: cause}

			got, err := f.find(actorCtx(tx))

			assertInternalWithCause(t, err, cause)
			if got == nil || !got.IsDefault() {
				t.Errorf("got %v, want actor.Default()", got)
			}
		})
	}
}

func TestActorQueriesUseEveryColumn(t *testing.T) {
	for _, query := range []string{selectActorByID, selectActorByUserID} {
		for _, column := range actorColumns {
			if !strings.Contains(query, column) {
				t.Errorf("query %q must select %q", query, column)
			}
		}
	}
	if !strings.Contains(selectActorByID, "WHERE id = $1") {
		t.Error("the id query must filter by id")
	}
	if !strings.Contains(selectActorByUserID, "WHERE user_id = $1") {
		t.Error("the user id query must filter by user id")
	}
	if strings.Count(upsertActor, "$") != len(actorColumns) {
		t.Error("the upsert must have one placeholder per column")
	}
}

func TestActorUpsertKeepsTheIdentityColumns(t *testing.T) {
	set := upsertActor[strings.Index(upsertActor, "DO UPDATE"):]

	for _, column := range []string{"id", "user_id", "created_at"} {
		if strings.Contains(set, column+" = EXCLUDED") {
			t.Errorf("the upsert must not overwrite %q", column)
		}
	}
	for _, column := range []string{"type", "name", "email", "status"} {
		if !strings.Contains(set, column+" = EXCLUDED."+column) {
			t.Errorf("the upsert must update %q", column)
		}
	}
}

func TestActorRepositorySaveInsertsTheActorInTheCurrentTransaction(t *testing.T) {
	tx := &actorTx{}
	a := repoActor()

	if err := saveActor(tx, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.execSQL != upsertActor {
		t.Fatal("expected the actor upsert")
	}
	if len(tx.execArgs) != len(actorColumns) {
		t.Fatalf("got %d args, want %d", len(tx.execArgs), len(actorColumns))
	}
	want := []any{a.ID(), "user_01", "producer", "Finca La Esperanza SAS", "user_01@example.com", "active"}
	for i := range want {
		if tx.execArgs[i] != want[i] {
			t.Errorf("arg %d: got %v, want %v", i, tx.execArgs[i], want[i])
		}
	}
	createdAt, ok := tx.execArgs[6].(time.Time)
	if !ok || !createdAt.Equal(a.CreatedAt()) {
		t.Errorf("created at arg: got %v, want %v", tx.execArgs[6], a.CreatedAt())
	}
}

func TestActorRepositorySaveRejectsADefaultActor(t *testing.T) {
	for _, a := range []*actor.Actor{actor.Default(), nil} {
		tx := &actorTx{}

		err := saveActor(tx, a)

		assertInternalWithCause(t, err, nil)
		if tx.execSQL != "" {
			t.Error("nothing must be saved for a default actor")
		}
	}
}

func TestActorRepositorySaveMapsTheUserIDViolationToAConflict(t *testing.T) {
	violation := &pgconn.PgError{Code: uniqueViolation, ConstraintName: actorUserIDConstraint}
	tx := &actorTx{execErr: violation}

	err := saveActor(tx, repoActor())

	if !exception.HasCode(err, exception.CodeActorAlreadyExists) {
		t.Errorf("got %v, want code %q", err, exception.CodeActorAlreadyExists)
	}
}

func TestActorRepositorySaveFailures(t *testing.T) {
	cause := errors.New("connection reset")
	otherViolation := &pgconn.PgError{Code: uniqueViolation, ConstraintName: "actor_pkey"}
	otherCode := &pgconn.PgError{Code: "23502", ConstraintName: actorUserIDConstraint}

	tests := []struct {
		name  string
		cause error
	}{
		{"connection failure", cause},
		{"violation of another constraint", otherViolation},
		{"another error on the user id constraint", otherCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := saveActor(&actorTx{execErr: tt.cause}, repoActor())

			assertInternalWithCause(t, err, tt.cause)
			if exception.HasCode(err, exception.CodeActorAlreadyExists) {
				t.Error("only a user id violation may be reported as a conflict")
			}
		})
	}
}
