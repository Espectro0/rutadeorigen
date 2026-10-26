package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.LotEventRepository = (*LotEventRepository)(nil)

type LotEventRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.LotEvent
}

func NewLotEventRepository(pool *pgxpool.Pool) *LotEventRepository {
	return &LotEventRepository{pool: pool, assembler: assembler.LotEvent{}}
}

func (r *LotEventRepository) Save(ctx context.Context, e *lot.Event) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("insert default lot event")
	}

	row, err := r.assembler.ToEntity(e)
	if err != nil {
		return err
	}

	_, err = conn(ctx, r.pool).Exec(ctx, insertLotEvent,
		row.ID, row.LotID, row.Sequence, row.Type, row.Stage, row.EventDate, row.Note, row.ActorID,
		row.ActorRole, row.Details, row.PreviousHash, row.Hash, row.RegisteredAt,
	)
	if isUniqueViolation(err, lotEventSequenceConstraint) {
		return exception.New(exception.CodeLotVersionConflict)
	}
	if err != nil {
		return internalError("insert lot event", err)
	}
	return nil
}

func (r *LotEventRepository) FindLast(ctx context.Context, lotID uuid.UUID) (*lot.Event, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectLastLotEvent, lotID)
	if err != nil {
		return lot.DefaultEvent(), internalError("query last lot event", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.LotEvent])
	if errors.Is(err, pgx.ErrNoRows) {
		return lot.DefaultEvent(), nil
	}
	if err != nil {
		return lot.DefaultEvent(), internalError("scan last lot event", err)
	}

	return r.assembler.ToDomain(row)
}

func (r *LotEventRepository) ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*lot.Event, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectLotEventsByLot, lotID)
	if err != nil {
		return nil, internalError("query lot events", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.LotEvent])
	if err != nil {
		return nil, internalError("scan lot events", err)
	}

	events := make([]*lot.Event, 0, len(entities))
	for _, row := range entities {
		e, err := r.assembler.ToDomain(row)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
