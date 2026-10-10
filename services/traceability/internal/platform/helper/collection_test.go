package helper_test

import (
	"encoding/json"
	"testing"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestSliceOrEmpty(t *testing.T) {
	t.Run("nil slice returns empty non-nil slice", func(t *testing.T) {
		got := helper.SliceOrEmpty[string](nil)
		if got == nil {
			t.Fatal("got nil, want empty slice")
		}
		if len(got) != 0 {
			t.Errorf("got length %d, want 0", len(got))
		}
	})

	t.Run("nil slice is encoded as [] in JSON", func(t *testing.T) {
		encoded, err := json.Marshal(helper.SliceOrEmpty[string](nil))
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != "[]" {
			t.Errorf("got %s, want []", encoded)
		}
	})

	t.Run("non-empty slice is returned as is", func(t *testing.T) {
		value := []string{"harvest", "drying"}
		got := helper.SliceOrEmpty(value)
		if len(got) != 2 || got[0] != "harvest" || got[1] != "drying" {
			t.Errorf("got %v, want %v", got, value)
		}
	})
}

func TestMapOrEmpty(t *testing.T) {
	t.Run("nil map returns empty usable map", func(t *testing.T) {
		got := helper.MapOrEmpty[string, int](nil)
		if got == nil {
			t.Fatal("got nil, want empty map")
		}
		got["key"] = 1
	})

	t.Run("non-empty map is returned as is", func(t *testing.T) {
		value := map[string]int{"altitude": 1650}
		got := helper.MapOrEmpty(value)
		if got["altitude"] != 1650 {
			t.Errorf("got %v, want %v", got, value)
		}
	})
}
