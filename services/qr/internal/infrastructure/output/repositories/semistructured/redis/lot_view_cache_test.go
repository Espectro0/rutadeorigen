package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/semistructured/redis"
	"rutadeorigen/qr/internal/platform/exception"
)

type fakeClient struct {
	goredis.Cmdable
	store   map[string][]byte
	ttl     time.Duration
	failing error
}

func newFake() *fakeClient {
	return &fakeClient{store: map[string][]byte{}}
}

func (f *fakeClient) Get(_ context.Context, key string) *goredis.StringCmd {
	if f.failing != nil {
		return goredis.NewStringResult("", f.failing)
	}
	raw, ok := f.store[key]
	if !ok {
		return goredis.NewStringResult("", goredis.Nil)
	}
	return goredis.NewStringResult(string(raw), nil)
}

func (f *fakeClient) Set(_ context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd {
	if f.failing != nil {
		return goredis.NewStatusResult("", f.failing)
	}
	f.store[key] = value.([]byte)
	f.ttl = expiration
	return goredis.NewStatusResult("OK", nil)
}

func (f *fakeClient) Del(_ context.Context, keys ...string) *goredis.IntCmd {
	if f.failing != nil {
		return goredis.NewIntResult(0, f.failing)
	}
	for _, k := range keys {
		delete(f.store, k)
	}
	return goredis.NewIntResult(int64(len(keys)), nil)
}

func newView(t *testing.T) *lotview.LotView {
	t.Helper()
	v, err := lotview.New(lotview.NewParams{
		LotID:        uuid.New(),
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
		CurrentStage: lotview.StagePackaging,
		Status:       lotview.StatusActive,
		Version:      3,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

func TestCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := newFake()
	cache := redis.NewLotViewCache(client, 0)
	view := newView(t)

	if err := cache.Set(ctx, view); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.store["qr:lot:"+view.LotID().String()]; !ok {
		t.Error("expected key qr:lot:{id}")
	}
	if client.ttl != redis.DefaultTTL {
		t.Errorf("expected default ttl, got %v", client.ttl)
	}

	got, err := cache.Get(ctx, view.LotID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsDefault() || got.LotID() != view.LotID() || got.Code() != "LT-001" ||
		got.Version() != 3 || got.CurrentStage() != lotview.StagePackaging ||
		got.FarmAltitude() != 1800 || !got.HarvestDate().Equal(view.HarvestDate()) {
		t.Errorf("unexpected cached view %+v", got)
	}
}

func TestCacheUsesConfiguredTTL(t *testing.T) {
	client := newFake()

	if err := redis.NewLotViewCache(client, time.Minute).Set(context.Background(), newView(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.ttl != time.Minute {
		t.Errorf("unexpected ttl %v", client.ttl)
	}
}

func TestCacheMissReturnsDefault(t *testing.T) {
	got, err := redis.NewLotViewCache(newFake(), 0).Get(context.Background(), uuid.New())

	if err != nil || !got.IsDefault() {
		t.Fatalf("expected default without error, got %v %v", got, err)
	}
}

func TestCacheDelete(t *testing.T) {
	ctx := context.Background()
	cache := redis.NewLotViewCache(newFake(), 0)
	view := newView(t)
	_ = cache.Set(ctx, view)

	if err := cache.Delete(ctx, view.LotID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := cache.Get(ctx, view.LotID())
	if !got.IsDefault() {
		t.Error("expected miss after delete")
	}
}

func TestCacheCorruptEntryIsAnError(t *testing.T) {
	client := newFake()
	id := uuid.New()
	client.store["qr:lot:"+id.String()] = []byte("{broken")

	got, err := redis.NewLotViewCache(client, 0).Get(context.Background(), id)

	if err == nil || !got.IsDefault() {
		t.Fatalf("expected error and default, got %v %v", got, err)
	}
}

func TestCacheWrapsClientFailures(t *testing.T) {
	ctx := context.Background()
	client := newFake()
	client.failing = errors.New("connection refused")
	cache := redis.NewLotViewCache(client, 0)
	view := newView(t)

	for name, err := range map[string]error{
		"set":    cache.Set(ctx, view),
		"delete": cache.Delete(ctx, view.LotID()),
		"get": func() error {
			_, err := cache.Get(ctx, view.LotID())
			return err
		}(),
	} {
		ex, ok := exception.From(err)
		if !ok || ex.Kind() != exception.KindInternal {
			t.Errorf("%s: expected internal exception, got %v", name, err)
		}
	}
}

func TestCacheRejectsDefaultView(t *testing.T) {
	if err := redis.NewLotViewCache(newFake(), 0).Set(context.Background(), lotview.Default()); err == nil {
		t.Error("expected error for default view")
	}
}
