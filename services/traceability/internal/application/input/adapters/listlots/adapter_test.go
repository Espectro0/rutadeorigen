package listlots_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/listlots"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

type fakeUseCase struct {
	page lot.Page
	err  error

	calls    int
	received listlotsuc.Query
}

func (f *fakeUseCase) Execute(_ context.Context, q listlotsuc.Query) (lot.Page, error) {
	f.calls++
	f.received = q
	if f.err != nil {
		return lot.Page{}, f.err
	}
	return f.page, nil
}

func sampleLot(minutes int) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageNone,
		Status:       lot.StatusActive,
		Version:      1,
		CreatedAt:    time.Date(2026, time.September, 20, 14, minutes, 0, 123456000, time.UTC),
	})
}

func encode(raw string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func TestAdapterBuildsTheQuery(t *testing.T) {
	ctx := context.Background()
	farmID, custodianID := uuid.New(), uuid.New()

	t.Run("maps filters and limit", func(t *testing.T) {
		uc := &fakeUseCase{}

		_, err := listlots.New(uc).Execute(ctx, request.ListLots{
			FarmID:      " " + farmID.String() + " ",
			CustodianID: custodianID.String(),
			Limit:       " 25 ",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := listlotsuc.Query{Filter: lot.Filter{FarmID: farmID, CustodianID: custodianID}, Limit: 25}
		if uc.received != want {
			t.Errorf("use case received %+v, want %+v", uc.received, want)
		}
	})

	t.Run("leaves everything unset when the request is empty", func(t *testing.T) {
		uc := &fakeUseCase{}

		if _, err := listlots.New(uc).Execute(ctx, request.ListLots{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if uc.received != (listlotsuc.Query{}) {
			t.Errorf("use case received %+v, want an empty query", uc.received)
		}
	})

	t.Run("rejects invalid input without calling the use case", func(t *testing.T) {
		tests := []struct {
			name string
			req  request.ListLots
			want exception.Code
		}{
			{"malformed farm id", request.ListLots{FarmID: "nope"}, exception.CodeLotInvalidFarmID},
			{"nil farm id", request.ListLots{FarmID: uuid.Nil.String()}, exception.CodeLotInvalidFarmID},
			{"malformed custodian id", request.ListLots{CustodianID: "nope"}, exception.CodeLotInvalidCustodianID},
			{"zero limit", request.ListLots{Limit: "0"}, exception.CodeInvalidLimit},
			{"negative limit", request.ListLots{Limit: "-5"}, exception.CodeInvalidLimit},
			{"non numeric limit", request.ListLots{Limit: "many"}, exception.CodeInvalidLimit},
			{"cursor that is not base64", request.ListLots{Cursor: "***"}, exception.CodeInvalidCursor},
			{"cursor without separator", request.ListLots{Cursor: encode("garbage")}, exception.CodeInvalidCursor},
			{"cursor with a bad timestamp", request.ListLots{Cursor: encode("abc." + uuid.NewString())}, exception.CodeInvalidCursor},
			{"cursor with a bad id", request.ListLots{Cursor: encode("1700000000000000000.nope")}, exception.CodeInvalidCursor},
			{"cursor with a nil id", request.ListLots{Cursor: encode("1700000000000000000." + uuid.Nil.String())}, exception.CodeInvalidCursor},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				uc := &fakeUseCase{}

				_, err := listlots.New(uc).Execute(ctx, tt.req)

				if !exception.HasCode(err, tt.want) {
					t.Errorf("got %v, want code %s", err, tt.want)
				}
				if uc.calls != 0 {
					t.Errorf("use case calls: got %d, want 0", uc.calls)
				}
			})
		}
	})
}

func TestAdapterBuildsThePage(t *testing.T) {
	ctx := context.Background()

	t.Run("maps the lots and encodes the next cursor", func(t *testing.T) {
		first, second := sampleLot(30), sampleLot(20)
		uc := &fakeUseCase{page: lot.Page{Lots: []*lot.Lot{first, second}, Next: lot.CursorOf(second)}}

		got, err := listlots.New(uc).Execute(ctx, request.ListLots{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Items) != 2 || got.Items[0].ID != first.ID().String() || got.Items[1].ID != second.ID().String() {
			t.Fatalf("items: got %+v", got.Items)
		}
		if got.NextCursor == "" {
			t.Fatal("NextCursor must be set when there are more lots")
		}
	})

	t.Run("the next cursor can be sent back and decodes to the same position", func(t *testing.T) {
		last := sampleLot(20)
		uc := &fakeUseCase{page: lot.Page{Lots: []*lot.Lot{last}, Next: lot.CursorOf(last)}}
		adapter := listlots.New(uc)

		page, err := adapter.Execute(ctx, request.ListLots{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := adapter.Execute(ctx, request.ListLots{Cursor: page.NextCursor}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		after := uc.received.After
		if after.ID != last.ID() || !after.CreatedAt.Equal(last.CreatedAt()) {
			t.Errorf("decoded cursor %+v, want the position of %s at %s", after, last.ID(), last.CreatedAt())
		}
	})

	t.Run("an empty page serializes as an empty array without a cursor", func(t *testing.T) {
		uc := &fakeUseCase{}

		got, err := listlots.New(uc).Execute(ctx, request.ListLots{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(raw) != `{"items":[]}` {
			t.Errorf("got %s, want {\"items\":[]}", raw)
		}
	})

	t.Run("returns the use case error", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		got, err := listlots.New(uc).Execute(ctx, request.ListLots{})

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if got.Items != nil || got.NextCursor != "" {
			t.Errorf("got %+v, want an empty response", got)
		}
	})
}
