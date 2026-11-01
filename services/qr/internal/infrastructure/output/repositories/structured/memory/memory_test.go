package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/memory"
)

func newView(t *testing.T, id uuid.UUID, version int) *lotview.LotView {
	t.Helper()
	v, err := lotview.New(lotview.NewParams{
		LotID:        id,
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageNone,
		Status:       lotview.StatusActive,
		Version:      version,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

func TestLotViewRepositoryMissReturnsDefault(t *testing.T) {
	repo := memory.NewLotViewRepository()

	got, err := repo.FindByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsDefault() {
		t.Error("expected default view on miss")
	}
}

func TestLotViewRepositorySaveKeepsNewestVersion(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewLotViewRepository()
	id := uuid.New()

	if err := repo.Save(ctx, newView(t, id, 2)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Save(ctx, newView(t, id, 1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := repo.FindByID(ctx, id)
	if got.Version() != 2 {
		t.Errorf("expected version 2 to be kept, got %d", got.Version())
	}

	if err := repo.Save(ctx, newView(t, id, 3)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ = repo.FindByID(ctx, id)
	if got.Version() != 3 {
		t.Errorf("expected version 3 to replace, got %d", got.Version())
	}
}

func TestQRCodeRepositorySaveIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewQRCodeRepository()
	id := uuid.New()

	missing, _ := repo.FindByLotID(ctx, id)
	if !missing.IsDefault() {
		t.Fatal("expected default code on miss")
	}

	first, err := qrcode.New(id, "https://example.com", qrcode.DefaultErrorCorrection, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := qrcode.New(id, "https://example.com", qrcode.DefaultErrorCorrection, "v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := repo.FindByLotID(ctx, id)
	if got.LibraryVersion() != "v1" {
		t.Errorf("expected first code to be kept, got %q", got.LibraryVersion())
	}
}

func TestRedirectRepositoryListsOnlyActive(t *testing.T) {
	rule, err := redirect.New("/l/", "/lotes/{id}", 301)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inactive := redirect.Restore(redirect.RestoreParams{
		ID:             uuid.New(),
		SourcePrefix:   "/old/",
		TargetTemplate: "/x/{id}",
		StatusCode:     302,
		Active:         false,
		CreatedAt:      time.Now(),
	})

	repo := memory.NewRedirectRepository(rule, inactive)

	rules, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 || rules[0].SourcePrefix() != "/l/" {
		t.Errorf("expected only the active rule, got %d", len(rules))
	}
}

func TestScanRepositoryStoresScans(t *testing.T) {
	repo := memory.NewScanRepository()

	s, err := scan.New(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Save(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Count() != 1 {
		t.Errorf("expected 1 scan, got %d", repo.Count())
	}
}

func TestProcessedEventRepositoryRegistersOnce(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewProcessedEventRepository()
	id := uuid.New()

	first, err := repo.Register(ctx, id, "lot.created")
	if err != nil || !first {
		t.Fatalf("expected first registration to succeed, got %v %v", first, err)
	}
	second, err := repo.Register(ctx, id, "lot.created")
	if err != nil || second {
		t.Fatalf("expected duplicate to be reported, got %v %v", second, err)
	}
}

func TestTxManagerRunsFunctionAndReturnsItsError(t *testing.T) {
	tx := memory.NewTxManager()
	boom := errors.New("boom")

	called := false
	err := tx.WithinTx(context.Background(), func(context.Context) error {
		called = true
		return boom
	})

	if !called {
		t.Error("expected function to run")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected the function error, got %v", err)
	}
}

func TestActorViewRepositorySavesAndFindsByUser(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewActorViewRepository()
	view, err := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "user_01", Role: actorview.RoleOrganization})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := repo.Save(ctx, view); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := repo.FindByUserID(ctx, "user_01")
	if err != nil || got != view {
		t.Errorf("expected the saved view, got %+v and %v", got, err)
	}
}

func TestActorViewRepositoryReturnsDefaultForUnknownUsers(t *testing.T) {
	got, err := memory.NewActorViewRepository().FindByUserID(context.Background(), "nobody")

	if err != nil || !got.IsDefault() {
		t.Errorf("expected the default view, got %+v and %v", got, err)
	}
}

func TestActorViewRepositorySaveReplacesTheViewOfTheSameUser(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewActorViewRepository()
	first, _ := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "user_01", Role: actorview.RoleProducer})
	second, _ := actorview.New(actorview.NewParams{ActorID: first.ActorID(), UserID: "user_01", Role: actorview.RoleBrand})

	_ = repo.Save(ctx, first)
	_ = repo.Save(ctx, second)

	got, _ := repo.FindByUserID(ctx, "user_01")
	if got.Role() != actorview.RoleBrand {
		t.Errorf("expected the latest view, got %+v", got)
	}
}

func TestActorViewRepositoryFindsByID(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewActorViewRepository()
	view, _ := actorview.New(actorview.NewParams{ActorID: uuid.New(), UserID: "user_01", Role: actorview.RoleRoaster})
	_ = repo.Save(ctx, view)

	got, err := repo.FindByID(ctx, view.ActorID())
	if err != nil || got != view {
		t.Errorf("expected the saved view, got %+v and %v", got, err)
	}
	if other, _ := repo.FindByID(ctx, uuid.New()); !other.IsDefault() {
		t.Error("an unknown id must return the default view")
	}
}

func TestScanRepositoryStatsByLot(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewScanRepository()
	lotID, otherLot := uuid.New(), uuid.New()
	day := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for _, s := range []*scan.Scan{
		scan.Restore(lotID, day.AddDate(0, 0, -40)),
		scan.Restore(lotID, day),
		scan.Restore(lotID, day.Add(2*time.Hour)),
		scan.Restore(lotID, day.AddDate(0, 0, 2)),
		scan.Restore(otherLot, day),
	} {
		_ = repo.Save(ctx, s)
	}

	got, err := repo.StatsByLot(ctx, lotID, day.AddDate(0, 0, -5))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.LotID() != lotID || got.Total() != 4 {
		t.Errorf("unexpected total %d for %s", got.Total(), got.LotID())
	}
	if !got.FirstScanAt().Equal(day.AddDate(0, 0, -40)) || !got.LastScanAt().Equal(day.AddDate(0, 0, 2)) {
		t.Errorf("unexpected first/last %v %v", got.FirstScanAt(), got.LastScanAt())
	}
	daily := got.Daily()
	if len(daily) != 2 || daily[0].Count != 2 || daily[1].Count != 1 || !daily[0].Day.Before(daily[1].Day) {
		t.Errorf("unexpected daily counts %+v", daily)
	}
}

func TestScanRepositoryStatsOfALotWithoutScans(t *testing.T) {
	got, err := memory.NewScanRepository().StatsByLot(context.Background(), uuid.New(), time.Now().AddDate(0, 0, -30))

	if err != nil || got.HasScans() || got.Daily() == nil || len(got.Daily()) != 0 {
		t.Errorf("expected empty stats, got %+v and %v", got, err)
	}
}
