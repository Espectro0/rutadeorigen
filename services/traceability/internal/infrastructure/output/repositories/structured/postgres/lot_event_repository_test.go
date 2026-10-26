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

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

var lotEventColumns = []string{
	"id", "lot_id", "sequence", "type", "stage", "event_date", "note", "actor_id",
	"actor_role", "details", "previous_hash", "hash", "registered_at",
}

type eventRows struct {
	pgx.Rows
	rows    []entity.LotEvent
	scanErr error
	index   int
}

func (r *eventRows) Close()     {}
func (r *eventRows) Err() error { return nil }

func (r *eventRows) FieldDescriptions() []pgconn.FieldDescription {
	fields := make([]pgconn.FieldDescription, len(lotEventColumns))
	for i, name := range lotEventColumns {
		fields[i] = pgconn.FieldDescription{Name: name}
	}
	return fields
}

func (r *eventRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *eventRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}

	e := r.rows[r.index-1]
	values := []any{
		e.ID, e.LotID, e.Sequence, e.Type, e.Stage, e.EventDate, e.Note, e.ActorID,
		e.ActorRole, e.Details, e.PreviousHash, e.Hash, e.RegisteredAt,
	}
	for i, target := range dest {
		switch p := target.(type) {
		case *uuid.UUID:
			*p = values[i].(uuid.UUID)
		case *int:
			*p = values[i].(int)
		case *string:
			*p = values[i].(string)
		case *time.Time:
			*p = values[i].(time.Time)
		case *[]byte:
			*p = values[i].([]byte)
		default:
			return errors.New("unsupported scan target")
		}
	}
	return nil
}

type eventTx struct {
	pgx.Tx
	execErr  error
	queryErr error
	rows     []entity.LotEvent
	scanErr  error

	execSQL   string
	execArgs  []any
	querySQL  string
	queryArgs []any
}

func (t *eventTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.execSQL, t.execArgs = sql, args
	return pgconn.CommandTag{}, t.execErr
}

func (t *eventTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	t.querySQL, t.queryArgs = sql, args
	if t.queryErr != nil {
		return nil, t.queryErr
	}
	return &eventRows{rows: t.rows, scanErr: t.scanErr}, nil
}

func eventCtx(tx *eventTx) context.Context {
	return context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
}

func newStageEventForRepo(t *testing.T, lotID uuid.UUID, previous *lot.Event, stage lot.Stage, details lot.StageDetails) *lot.Event {
	t.Helper()

	e, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       lotID,
		HarvestDate: time.Now().AddDate(0, -1, 0),
		Previous:    previous,
		Stage:       stage,
		EventDate:   time.Now().AddDate(0, 0, -1),
		Note:        "Dried on raised beds",
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details:     details,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func repoFarmDetails() lot.FarmDetails {
	return lot.FarmDetails{
		DryingMethod:      lot.DryingMethodSunPatio,
		DryingDays:        15,
		MoisturePercent:   11.5,
		ParchmentWeightKg: 120.5,
	}
}

func repoRoastingDetails() lot.RoastingDetails {
	return lot.RoastingDetails{
		RoastLevel:      lot.RoastLevelMedium,
		DurationMinutes: 14,
		GreenWeightKg:   100,
		RoastedWeightKg: 84,
	}
}

func toRow(t *testing.T, e *lot.Event) entity.LotEvent {
	t.Helper()

	row, err := assembler.LotEvent{}.ToEntity(e)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func saveEvent(tx *eventTx, e *lot.Event) error {
	return NewLotEventRepository(nil).Save(eventCtx(tx), e)
}

func assertInternalWithCause(t *testing.T, err, cause error) {
	t.Helper()

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("got %v, want an internal error", err)
	}
	if cause != nil && !errors.Is(err, cause) {
		t.Error("expected the original cause to be kept")
	}
}

func TestLotEventQueriesOrderTheEvents(t *testing.T) {
	if !strings.Contains(selectLastLotEvent, "ORDER BY sequence DESC") || !strings.Contains(selectLastLotEvent, "LIMIT 1") {
		t.Error("the last event query must take the highest sequence")
	}
	if !strings.Contains(selectLotEventsByLot, "ORDER BY sequence ASC") {
		t.Error("the list query must be ordered by ascending sequence")
	}
	for _, query := range []string{selectLastLotEvent, selectLotEventsByLot} {
		if !strings.Contains(query, "WHERE lot_id = $1") {
			t.Error("the queries must filter by lot")
		}
	}
}

func TestLotEventRepositorySaveInsertsTheEventInTheCurrentTransaction(t *testing.T) {
	tx := &eventTx{}
	e := newStageEventForRepo(t, uuid.New(), nil, lot.StageFarm, repoFarmDetails())

	if err := saveEvent(tx, e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.execSQL != insertLotEvent {
		t.Fatal("expected the lot event insert")
	}
	row := toRow(t, e)
	want := []any{
		row.ID, row.LotID, 1, "stage", "farm", row.EventDate, "Dried on raised beds", row.ActorID,
		"producer", row.Details, lot.GenesisHash, e.Hash(), row.RegisteredAt,
	}
	if len(tx.execArgs) != len(want) {
		t.Fatalf("got %d args, want %d", len(tx.execArgs), len(want))
	}
	for i := range want {
		if i == 9 {
			continue
		}
		if tx.execArgs[i] != want[i] {
			t.Errorf("arg %d: got %v, want %v", i, tx.execArgs[i], want[i])
		}
	}
	details, ok := tx.execArgs[9].([]byte)
	if !ok || string(details) != string(row.Details) {
		t.Errorf("details arg: got %v, want %s", tx.execArgs[9], row.Details)
	}
	if strings.Count(insertLotEvent, "$") != len(lotEventColumns) {
		t.Error("the insert must have one placeholder per column")
	}
}

func TestLotEventRepositorySaveRejectsADefaultEvent(t *testing.T) {
	for _, e := range []*lot.Event{lot.DefaultEvent(), nil} {
		tx := &eventTx{}

		err := saveEvent(tx, e)

		assertInternalWithCause(t, err, nil)
		if tx.execSQL != "" {
			t.Error("nothing must be inserted for a default event")
		}
	}
}

func TestLotEventRepositorySaveRejectsAnEventWithoutDetails(t *testing.T) {
	tx := &eventTx{}
	e := lot.RestoreEvent(lot.RestoreEventParams{
		ID:        uuid.New(),
		LotID:     uuid.New(),
		Sequence:  1,
		EventType: lot.EventTypeStage,
		Stage:     lot.StageFarm,
	})

	err := saveEvent(tx, e)

	assertInternalWithCause(t, err, nil)
	if tx.execSQL != "" {
		t.Error("nothing must be inserted for an event without details")
	}
}

func TestLotEventRepositorySaveMapsTheSequenceViolationToAConflict(t *testing.T) {
	violation := &pgconn.PgError{Code: uniqueViolation, ConstraintName: lotEventSequenceConstraint}
	tx := &eventTx{execErr: violation}

	err := saveEvent(tx, newStageEventForRepo(t, uuid.New(), nil, lot.StageFarm, repoFarmDetails()))

	if !exception.HasCode(err, exception.CodeLotVersionConflict) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotVersionConflict)
	}
}

func TestLotEventRepositorySaveFailures(t *testing.T) {
	cause := errors.New("connection reset")
	otherViolation := &pgconn.PgError{Code: uniqueViolation, ConstraintName: "lot_event_hash_key"}

	tests := []struct {
		name  string
		cause error
	}{
		{"the insert fails", cause},
		{"another unique constraint is violated", otherViolation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &eventTx{execErr: tt.cause}

			err := saveEvent(tx, newStageEventForRepo(t, uuid.New(), nil, lot.StageFarm, repoFarmDetails()))

			assertInternalWithCause(t, err, tt.cause)
		})
	}
}

func TestLotEventRepositoryFindLast(t *testing.T) {
	lotID := uuid.New()
	repo := NewLotEventRepository(nil)

	t.Run("queries the last event of the lot in the current transaction", func(t *testing.T) {
		tx := &eventTx{}

		if _, err := repo.FindLast(eventCtx(tx), lotID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tx.querySQL != selectLastLotEvent || len(tx.queryArgs) != 1 || tx.queryArgs[0] != lotID {
			t.Error("expected the last event query for the lot")
		}
	})

	t.Run("returns the event with a hash that still verifies", func(t *testing.T) {
		stored := newStageEventForRepo(t, lotID, nil, lot.StageFarm, repoFarmDetails())
		tx := &eventTx{rows: []entity.LotEvent{toRow(t, stored)}}

		got, err := repo.FindLast(eventCtx(tx), lotID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.ID() != stored.ID() || got.Sequence() != 1 || got.Stage() != lot.StageFarm {
			t.Errorf("unexpected event: %s, sequence %d, stage %q", got.ID(), got.Sequence(), got.Stage())
		}
		if got.Details() != lot.StageDetails(repoFarmDetails()) {
			t.Errorf("Details: got %+v", got.Details())
		}
		if !got.VerifyHash() {
			t.Error("the loaded event must keep a verifiable hash")
		}
	})

	t.Run("a lot without events returns the default event", func(t *testing.T) {
		got, err := repo.FindLast(eventCtx(&eventTx{}), lotID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.IsDefault() {
			t.Error("expected the default event")
		}
	})

	t.Run("the query fails", func(t *testing.T) {
		cause := errors.New("connection reset")

		got, err := repo.FindLast(eventCtx(&eventTx{queryErr: cause}), lotID)

		assertInternalWithCause(t, err, cause)
		if !got.IsDefault() {
			t.Error("expected the default event on error")
		}
	})

	t.Run("scanning the row fails", func(t *testing.T) {
		cause := errors.New("bad row")
		tx := &eventTx{rows: []entity.LotEvent{{}}, scanErr: cause}

		got, err := repo.FindLast(eventCtx(tx), lotID)

		assertInternalWithCause(t, err, cause)
		if !got.IsDefault() {
			t.Error("expected the default event on error")
		}
	})

	t.Run("corrupt details are reported", func(t *testing.T) {
		row := toRow(t, newStageEventForRepo(t, lotID, nil, lot.StageFarm, repoFarmDetails()))
		row.Details = []byte(`{"unexpected":true}`)

		got, err := repo.FindLast(eventCtx(&eventTx{rows: []entity.LotEvent{row}}), lotID)

		assertInternalWithCause(t, err, nil)
		if !got.IsDefault() {
			t.Error("expected the default event on error")
		}
	})
}

func TestLotEventRepositoryListByLotID(t *testing.T) {
	lotID := uuid.New()
	repo := NewLotEventRepository(nil)

	t.Run("queries the events of the lot in the current transaction", func(t *testing.T) {
		tx := &eventTx{}

		if _, err := repo.ListByLotID(eventCtx(tx), lotID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tx.querySQL != selectLotEventsByLot || len(tx.queryArgs) != 1 || tx.queryArgs[0] != lotID {
			t.Error("expected the events query for the lot")
		}
	})

	t.Run("returns the events in the order of the rows with a valid chain", func(t *testing.T) {
		farm := newStageEventForRepo(t, lotID, nil, lot.StageFarm, repoFarmDetails())
		roasting := newStageEventForRepo(t, lotID, farm, lot.StageRoasting, repoRoastingDetails())
		tx := &eventTx{rows: []entity.LotEvent{toRow(t, farm), toRow(t, roasting)}}

		got, err := repo.ListByLotID(eventCtx(tx), lotID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 2 || got[0].Sequence() != 1 || got[1].Sequence() != 2 {
			t.Fatalf("unexpected events: %v", got)
		}
		if got[0].Stage() != lot.StageFarm || got[1].Stage() != lot.StageRoasting {
			t.Error("unexpected stages")
		}
		if got[1].PreviousHash() != got[0].Hash() {
			t.Error("the hash chain is broken")
		}
		for i, e := range got {
			if !e.VerifyHash() {
				t.Errorf("event %d does not verify", i+1)
			}
		}
	})

	t.Run("a lot without events returns an empty list", func(t *testing.T) {
		got, err := repo.ListByLotID(eventCtx(&eventTx{}), lotID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("got %v, want an empty non-nil list", got)
		}
	})

	t.Run("the query fails", func(t *testing.T) {
		cause := errors.New("connection reset")

		got, err := repo.ListByLotID(eventCtx(&eventTx{queryErr: cause}), lotID)

		assertInternalWithCause(t, err, cause)
		if got != nil {
			t.Error("expected no events on error")
		}
	})

	t.Run("scanning a row fails", func(t *testing.T) {
		cause := errors.New("bad row")
		tx := &eventTx{rows: []entity.LotEvent{{}}, scanErr: cause}

		got, err := repo.ListByLotID(eventCtx(tx), lotID)

		assertInternalWithCause(t, err, cause)
		if got != nil {
			t.Error("expected no events on error")
		}
	})

	t.Run("a row with corrupt details fails the whole list", func(t *testing.T) {
		good := toRow(t, newStageEventForRepo(t, lotID, nil, lot.StageFarm, repoFarmDetails()))
		bad := good
		bad.ID = uuid.New()
		bad.Sequence = 2
		bad.Details = []byte(`not json`)

		got, err := repo.ListByLotID(eventCtx(&eventTx{rows: []entity.LotEvent{good, bad}}), lotID)

		assertInternalWithCause(t, err, nil)
		if got != nil {
			t.Error("expected no events on error")
		}
	})
}
