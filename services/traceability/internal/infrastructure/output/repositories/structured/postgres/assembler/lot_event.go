package assembler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

type LotEvent struct{}

func (LotEvent) ToDomain(e entity.LotEvent) (*lot.Event, error) {
	details, err := decodeStageDetails(lot.ParseStage(e.Stage), e.Details)
	if err != nil {
		return lot.DefaultEvent(), err
	}

	return lot.RestoreEvent(lot.RestoreEventParams{
		ID:           e.ID,
		LotID:        e.LotID,
		Sequence:     e.Sequence,
		EventType:    lot.ParseEventType(e.Type),
		Stage:        lot.ParseStage(e.Stage),
		EventDate:    e.EventDate.UTC(),
		Note:         e.Note,
		ActorID:      e.ActorID,
		ActorRole:    e.ActorRole,
		Details:      details,
		PreviousHash: e.PreviousHash,
		Hash:         e.Hash,
		RegisteredAt: e.RegisteredAt.UTC(),
	}), nil
}

func (LotEvent) ToEntity(e *lot.Event) (entity.LotEvent, error) {
	if e.IsDefault() {
		return entity.LotEvent{}, nil
	}
	if e.Details() == nil {
		return entity.LotEvent{}, exception.New(exception.CodeInternal).
			WithTechnical("lot event without details")
	}

	details, err := json.Marshal(e.Details())
	if err != nil {
		return entity.LotEvent{}, exception.New(exception.CodeInternal).
			WithTechnical("encode lot event details").
			WithCause(err)
	}

	return entity.LotEvent{
		ID:           e.ID(),
		LotID:        e.LotID(),
		Sequence:     e.Sequence(),
		Type:         e.Type().String(),
		Stage:        e.Stage().String(),
		EventDate:    e.EventDate().UTC(),
		Note:         e.Note(),
		ActorID:      e.ActorID(),
		ActorRole:    e.ActorRole(),
		Details:      details,
		PreviousHash: e.PreviousHash(),
		Hash:         e.Hash(),
		RegisteredAt: e.RegisteredAt().UTC(),
	}, nil
}

func decodeStageDetails(stage lot.Stage, raw []byte) (lot.StageDetails, error) {
	switch stage {
	case lot.StageFarm:
		return decodeStrict[lot.FarmDetails](raw)
	case lot.StageRoasting:
		return decodeStrict[lot.RoastingDetails](raw)
	case lot.StagePackaging:
		return decodeStrict[lot.PackagingDetails](raw)
	default:
		return nil, exception.New(exception.CodeInternal).
			WithTechnical("unsupported lot event stage: " + stage.String())
	}
}

func decodeStrict[T lot.StageDetails](raw []byte) (lot.StageDetails, error) {
	var details T

	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, exception.New(exception.CodeInternal).
			WithTechnical("lot event details are null")
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&details); err != nil {
		return nil, exception.New(exception.CodeInternal).
			WithTechnical("decode lot event details").
			WithCause(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, exception.New(exception.CodeInternal).
			WithTechnical("lot event details have trailing data")
	}
	return details, nil
}
