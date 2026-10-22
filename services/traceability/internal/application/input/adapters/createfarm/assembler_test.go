package createfarm_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/createfarm"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/farm"
)

func validRequest() request.CreateFarm {
	return request.CreateFarm{Name: "La Esperanza", Altitude: 1650}
}

func TestAssemblerToDomain(t *testing.T) {
	t.Run("builds a draft with the request values", func(t *testing.T) {
		draft, err := createfarm.Assembler{}.ToDomain(validRequest())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if draft.Name() != "La Esperanza" || draft.Altitude() != 1650 {
			t.Errorf("got %q %d, want %q %d", draft.Name(), draft.Altitude(), "La Esperanza", 1650)
		}
	})

	t.Run("draft is not a persisted farm", func(t *testing.T) {
		draft, _ := createfarm.Assembler{}.ToDomain(validRequest())

		if !draft.IsDefault() {
			t.Error("a draft must not have an ID yet")
		}
		if draft.ProducerID() != uuid.Nil {
			t.Error("a draft must not have a producer yet")
		}
	})

	t.Run("does not validate: the domain does", func(t *testing.T) {
		draft, err := createfarm.Assembler{}.ToDomain(request.CreateFarm{Name: "  x ", Altitude: -5})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if draft.Name() != "  x " || draft.Altitude() != -5 {
			t.Error("the draft must keep the raw values for the domain to validate")
		}
	})
}

func TestAssemblerToDTO(t *testing.T) {
	t.Run("maps the farm to a response", func(t *testing.T) {
		f := farm.Restore(uuid.New(), uuid.New(), "La Esperanza", 1650, time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC))

		got := createfarm.Assembler{}.ToDTO(f)

		if got.ID != f.ID().String() || got.Name != "La Esperanza" || got.Altitude != 1650 {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("default farm returns an empty response", func(t *testing.T) {
		if got := (createfarm.Assembler{}).ToDTO(farm.Default()); got != (response.Farm{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})
}
