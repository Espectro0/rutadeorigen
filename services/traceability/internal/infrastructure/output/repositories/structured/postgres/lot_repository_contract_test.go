package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/repotest"
)

func TestLotRepositoryContract(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := postgres.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	seed := func(t *testing.T, farmID, custodianID uuid.UUID) {
		t.Helper()

		_, err := pool.Exec(ctx, `
			INSERT INTO actor (id, user_id, type, name, status, created_at)
			VALUES ($1, $2, 'producer', 'Test producer', 'active', now())
			ON CONFLICT (id) DO NOTHING`, custodianID, custodianID.String())
		if err != nil {
			t.Fatalf("seed actor: %v", err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO farm (id, producer_id, name, altitude, created_at)
			VALUES ($1, $2, 'Test farm', 1500, now())
			ON CONFLICT (id) DO NOTHING`, farmID, custodianID)
		if err != nil {
			t.Fatalf("seed farm: %v", err)
		}
	}

	repotest.RunLotRepositoryContract(t, postgres.NewLotRepository(pool), seed)
}
