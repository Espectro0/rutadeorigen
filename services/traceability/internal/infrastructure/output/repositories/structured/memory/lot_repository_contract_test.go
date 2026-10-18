package memory_test

import (
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/repotest"
)

func TestLotRepositoryContract(t *testing.T) {
	seed := func(*testing.T, uuid.UUID, uuid.UUID) {}

	repotest.RunLotRepositoryContract(t, memory.NewLotRepository(), seed)
}
