package lot_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var sha256HexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validStageEventParams() lot.StageEventParams {
	return lot.StageEventParams{
		LotID:       uuid.New(),
		HarvestDate: validHarvestDate(),
		Stage:       lot.StageFarm,
		EventDate:   helper.DateUTC(time.Now().AddDate(0, 0, -1)),
		Note:        "Dried on raised beds",
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details:     validFarmDetails(),
	}
}

func newValidStageEvent(t *testing.T) *lot.LotEvent {
	t.Helper()
	e, err := lot.NewStageEvent(validStageEventParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return e
}

func restoreParamsOf(e *lot.LotEvent) lot.RestoreEventParams {
	return lot.RestoreEventParams{
		ID:           e.ID(),
		LotID:        e.LotID(),
		Sequence:     e.Sequence(),
		EventType:    e.Type(),
		Stage:        e.Stage(),
		EventDate:    e.EventDate(),
		Note:         e.Note(),
		ActorID:      e.ActorID(),
		ActorRole:    e.ActorRole(),
		Details:      e.Details(),
		PreviousHash: e.PreviousHash(),
		Hash:         e.Hash(),
		RegisteredAt: e.RegisteredAt(),
	}
}

func TestNewStageEventFirstEvent(t *testing.T) {
	params := validStageEventParams()
	params.Note = "  Dried on raised beds  "
	params.ActorRole = "  producer  "

	before := time.Now().UTC().Add(-time.Second)
	e, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.IsDefault() || e.ID() == uuid.Nil {
		t.Error("expected a generated ID")
	}
	if e.LotID() != params.LotID || e.ActorID() != params.ActorID {
		t.Error("unexpected lot or actor")
	}
	if e.Sequence() != 1 {
		t.Errorf("expected sequence 1, got %d", e.Sequence())
	}
	if e.PreviousHash() != lot.GenesisHash {
		t.Errorf("expected the genesis hash, got %q", e.PreviousHash())
	}
	if e.Type() != lot.EventTypeStage || e.Stage() != lot.StageFarm {
		t.Errorf("unexpected type %q or stage %q", e.Type(), e.Stage())
	}
	if e.Note() != "Dried on raised beds" || e.ActorRole() != "producer" {
		t.Errorf("expected trimmed values, got %q and %q", e.Note(), e.ActorRole())
	}
	if e.Details() == nil || e.Details().Stage() != lot.StageFarm {
		t.Error("expected the farm details")
	}
	if !sha256HexPattern.MatchString(e.Hash()) {
		t.Errorf("expected a sha256 hex hash, got %q", e.Hash())
	}
	if !e.VerifyHash() {
		t.Error("expected the hash to verify")
	}
	if e.RegisteredAt().Before(before) || e.RegisteredAt().After(time.Now().UTC().Add(time.Second)) {
		t.Errorf("unexpected registration time %v", e.RegisteredAt())
	}
	if e.RegisteredAt().Location() != time.UTC {
		t.Error("expected the registration time in UTC")
	}
	if !e.RegisteredAt().Equal(e.RegisteredAt().Truncate(time.Microsecond)) {
		t.Error("expected the registration time truncated to microseconds")
	}
}

func TestGenesisHash(t *testing.T) {
	if len(lot.GenesisHash) != 64 || strings.Trim(lot.GenesisHash, "0") != "" {
		t.Errorf("expected 64 zeros, got %q", lot.GenesisHash)
	}
}

func TestNewStageEventChain(t *testing.T) {
	first := newValidStageEvent(t)

	roasting := validStageEventParams()
	roasting.LotID = first.LotID()
	roasting.Previous = first
	roasting.Stage = lot.StageRoasting
	roasting.Details = validRoastingDetails()

	second, err := lot.NewStageEvent(roasting)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.Sequence() != 2 {
		t.Errorf("expected sequence 2, got %d", second.Sequence())
	}
	if second.PreviousHash() != first.Hash() {
		t.Errorf("expected previous hash %q, got %q", first.Hash(), second.PreviousHash())
	}
	if second.Hash() == first.Hash() || !second.VerifyHash() {
		t.Error("expected a distinct hash that verifies")
	}

	packaging := validStageEventParams()
	packaging.LotID = first.LotID()
	packaging.Previous = second
	packaging.Stage = lot.StagePackaging
	packaging.Details = validPackagingDetails()

	third, err := lot.NewStageEvent(packaging)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if third.Sequence() != 3 || third.PreviousHash() != second.Hash() || !third.VerifyHash() {
		t.Errorf("unexpected third event: sequence %d, previous %q", third.Sequence(), third.PreviousHash())
	}
}

func TestNewStageEventDefaultPreviousStartsTheChain(t *testing.T) {
	for name, previous := range map[string]*lot.LotEvent{"nil": nil, "default": lot.DefaultLotEvent()} {
		t.Run(name, func(t *testing.T) {
			params := validStageEventParams()
			params.Previous = previous

			e, err := lot.NewStageEvent(params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Sequence() != 1 || e.PreviousHash() != lot.GenesisHash {
				t.Errorf("unexpected position: sequence %d, previous %q", e.Sequence(), e.PreviousHash())
			}
		})
	}
}

func TestNewStageEventPreviousFromAnotherLot(t *testing.T) {
	params := validStageEventParams()
	params.Previous = newValidStageEvent(t)

	e, err := lot.NewStageEvent(params)
	if !exception.HasCode(err, exception.CodeInternal) {
		t.Fatalf("got %v, want code %q", err, exception.CodeInternal)
	}
	if !e.IsDefault() {
		t.Error("expected the default event on error")
	}
}

func TestNewStageEventNormalizesEventDateToUTCDate(t *testing.T) {
	params := validStageEventParams()
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	params.EventDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 15, 30, 45, 0, time.UTC)

	e, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	if !e.EventDate().Equal(want) {
		t.Errorf("event date = %v, want %v", e.EventDate(), want)
	}
}

func TestNewStageEventEventDateBoundaries(t *testing.T) {
	harvest := validHarvestDate()
	today := helper.DateUTC(time.Now())

	tests := []struct {
		name      string
		eventDate time.Time
		wantCode  exception.Code
	}{
		{"on the harvest date", harvest, ""},
		{"day after the harvest date", harvest.AddDate(0, 0, 1), ""},
		{"day before the harvest date", harvest.AddDate(0, 0, -1), exception.CodeLotInvalidEventDate},
		{"today", today, ""},
		{"tomorrow", today.AddDate(0, 0, 1), exception.CodeLotInvalidEventDate},
		{"default date", time.Time{}, exception.CodeLotInvalidEventDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validStageEventParams()
			params.HarvestDate = harvest
			params.EventDate = tt.eventDate

			e, err := lot.NewStageEvent(params)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !exception.HasCode(err, tt.wantCode) {
				t.Fatalf("got %v, want code %q", err, tt.wantCode)
			}
			if !e.IsDefault() {
				t.Error("expected the default event on error")
			}
		})
	}
}

func TestNewStageEventNoteLength(t *testing.T) {
	params := validStageEventParams()

	params.Note = strings.Repeat("ñ", 500)
	if _, err := lot.NewStageEvent(params); err != nil {
		t.Errorf("500 characters: unexpected error: %v", err)
	}

	params.Note = ""
	if _, err := lot.NewStageEvent(params); err != nil {
		t.Errorf("empty note: unexpected error: %v", err)
	}

	params.Note = strings.Repeat("ñ", 501)
	_, err := lot.NewStageEvent(params)
	if !exception.HasCode(err, exception.CodeLotInvalidNote) {
		t.Fatalf("501 characters: got %v, want code %q", err, exception.CodeLotInvalidNote)
	}
	ex, _ := exception.From(err)
	if ex.Params()["max"] != 500 {
		t.Errorf("params = %v, want max=500", ex.Params())
	}
}

func TestNewStageEventInvalid(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*lot.StageEventParams)
		wantCode exception.Code
	}{
		{"default lot ID", func(p *lot.StageEventParams) { p.LotID = uuid.Nil }, exception.CodeInternal},
		{"default actor ID", func(p *lot.StageEventParams) { p.ActorID = uuid.Nil }, exception.CodeInternal},
		{"empty actor role", func(p *lot.StageEventParams) { p.ActorRole = "   " }, exception.CodeInternal},
		{"stage none", func(p *lot.StageEventParams) { p.Stage = lot.StageNone }, exception.CodeLotInvalidStage},
		{"stage unknown", func(p *lot.StageEventParams) { p.Stage = lot.StageUnknown }, exception.CodeLotInvalidStage},
		{"stage not recognized", func(p *lot.StageEventParams) { p.Stage = lot.Stage("bogus") }, exception.CodeLotInvalidStage},
		{"nil details", func(p *lot.StageEventParams) { p.Details = nil }, exception.CodeStageInvalidDetails},
		{"details from another stage", func(p *lot.StageEventParams) { p.Details = validRoastingDetails() }, exception.CodeStageInvalidDetails},
		{"packaging stage with farm details", func(p *lot.StageEventParams) { p.Stage = lot.StagePackaging }, exception.CodeStageInvalidDetails},
		{"invalid farm details", func(p *lot.StageEventParams) { p.Details = lot.FarmDetails{} }, exception.CodeStageInvalidDryingMethod},
		{"invalid roasting details", func(p *lot.StageEventParams) {
			p.Stage = lot.StageRoasting
			p.Details = lot.RoastingDetails{}
		}, exception.CodeStageInvalidRoastLevel},
		{"packaging best before not after the event date", func(p *lot.StageEventParams) {
			p.Stage = lot.StagePackaging
			details := validPackagingDetails()
			details.BestBefore = p.EventDate.Format(time.DateOnly)
			p.Details = details
		}, exception.CodeStageInvalidBestBefore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validStageEventParams()
			tt.mutate(&params)

			e, err := lot.NewStageEvent(params)
			if !exception.HasCode(err, tt.wantCode) {
				t.Fatalf("got %v, want code %q", err, tt.wantCode)
			}
			if !e.IsDefault() {
				t.Error("expected the default event on error")
			}
		})
	}
}

func TestNewStageEventPackagingWithValidDetails(t *testing.T) {
	params := validStageEventParams()
	params.Stage = lot.StagePackaging
	params.Details = validPackagingDetails()

	e, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Stage() != lot.StagePackaging || !e.VerifyHash() {
		t.Error("expected a verifiable packaging event")
	}
}

func TestRestoreEventRoundTrip(t *testing.T) {
	original := newValidStageEvent(t)

	restored := lot.RestoreEvent(restoreParamsOf(original))

	if restored.IsDefault() || restored.ID() != original.ID() || restored.LotID() != original.LotID() {
		t.Error("unexpected identity")
	}
	if restored.Sequence() != original.Sequence() || restored.Type() != original.Type() || restored.Stage() != original.Stage() {
		t.Error("unexpected position or classification")
	}
	if !restored.EventDate().Equal(original.EventDate()) || !restored.RegisteredAt().Equal(original.RegisteredAt()) {
		t.Error("unexpected dates")
	}
	if restored.Note() != original.Note() || restored.ActorID() != original.ActorID() || restored.ActorRole() != original.ActorRole() {
		t.Error("unexpected note or actor")
	}
	if restored.PreviousHash() != original.PreviousHash() || restored.Hash() != original.Hash() {
		t.Error("unexpected hashes")
	}
	if !restored.VerifyHash() {
		t.Error("expected the restored event to verify")
	}
}

func TestVerifyHashDetectsTampering(t *testing.T) {
	original := newValidStageEvent(t)

	tests := []struct {
		name   string
		mutate func(*lot.RestoreEventParams)
	}{
		{"note", func(p *lot.RestoreEventParams) { p.Note = "Edited afterwards" }},
		{"actor role", func(p *lot.RestoreEventParams) { p.ActorRole = "admin" }},
		{"actor ID", func(p *lot.RestoreEventParams) { p.ActorID = uuid.New() }},
		{"lot ID", func(p *lot.RestoreEventParams) { p.LotID = uuid.New() }},
		{"sequence", func(p *lot.RestoreEventParams) { p.Sequence = 7 }},
		{"stage", func(p *lot.RestoreEventParams) { p.Stage = lot.StageRoasting }},
		{"event type", func(p *lot.RestoreEventParams) { p.EventType = lot.EventTypeCorrection }},
		{"event date", func(p *lot.RestoreEventParams) { p.EventDate = p.EventDate.AddDate(0, 0, -1) }},
		{"registration time", func(p *lot.RestoreEventParams) { p.RegisteredAt = p.RegisteredAt.Add(time.Microsecond) }},
		{"previous hash", func(p *lot.RestoreEventParams) { p.PreviousHash = strings.Repeat("a", 64) }},
		{"details", func(p *lot.RestoreEventParams) {
			details := validFarmDetails()
			details.DryingDays++
			p.Details = details
		}},
		{"hash", func(p *lot.RestoreEventParams) { p.Hash = strings.Repeat("b", 64) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := restoreParamsOf(original)
			tt.mutate(&params)

			if lot.RestoreEvent(params).VerifyHash() {
				t.Error("expected the tampered event not to verify")
			}
		})
	}
}

func TestVerifyHashIgnoresEventTimeOfDayAndLocation(t *testing.T) {
	original := newValidStageEvent(t)

	params := restoreParamsOf(original)
	params.EventDate = original.EventDate().In(time.FixedZone("UTC-5", -5*3600))
	params.RegisteredAt = original.RegisteredAt().In(time.FixedZone("UTC+2", 2*3600))

	if !lot.RestoreEvent(params).VerifyHash() {
		t.Error("expected the hash to be independent of the time zone")
	}
}

func TestComputeHashIsDeterministic(t *testing.T) {
	e := newValidStageEvent(t)

	if e.ComputeHash() != e.ComputeHash() {
		t.Error("expected the same hash on every call")
	}
	if e.ComputeHash() != e.Hash() {
		t.Error("expected the stored hash to match the computed one")
	}
}

func TestComputeHashDiffersBetweenEvents(t *testing.T) {
	params := validStageEventParams()

	a, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params.Note = "Another note"
	b, err := lot.NewStageEvent(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Hash() == b.Hash() {
		t.Error("expected distinct hashes for events with different content")
	}
}

func TestVerifyHashOfDefaultEvents(t *testing.T) {
	var nilEvent *lot.LotEvent

	if lot.DefaultLotEvent().VerifyHash() {
		t.Error("the default event must not verify")
	}
	if nilEvent.VerifyHash() {
		t.Error("a nil event must not verify")
	}
}

func TestDefaultLotEvent(t *testing.T) {
	e := lot.DefaultLotEvent()

	if !e.IsDefault() {
		t.Error("expected the default event")
	}
	if e.Sequence() != 0 || e.Type() != lot.EventTypeUnknown || e.Stage() != lot.StageUnknown {
		t.Errorf("unexpected defaults: sequence %d, type %q, stage %q", e.Sequence(), e.Type(), e.Stage())
	}
	if e.Details() != nil || e.Note() != "" || e.Hash() != "" || e.PreviousHash() != "" {
		t.Error("expected empty details, note and hashes")
	}
	if lot.DefaultLotEvent() == e {
		t.Error("expected an independent instance on every call")
	}
}

func TestLotEventIsDefaultNilSafe(t *testing.T) {
	var e *lot.LotEvent
	if !e.IsDefault() {
		t.Error("a nil event must be default")
	}
	if newValidStageEvent(t).IsDefault() {
		t.Error("a valid event must not be default")
	}
}
