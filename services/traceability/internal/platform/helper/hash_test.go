package helper_test

import (
	"testing"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestSHA256Hex(t *testing.T) {
	t.Run("known value", func(t *testing.T) {
		want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		if got := helper.SHA256Hex([]byte("abc")); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("same data gives same hash", func(t *testing.T) {
		data := []byte("lot-1|harvest|2026-10-09")

		first := helper.SHA256Hex(data)
		second := helper.SHA256Hex(data)

		if first != second {
			t.Error("expected the hash to be deterministic")
		}
	})

	t.Run("one character change gives a different hash", func(t *testing.T) {
		first := helper.SHA256Hex([]byte("lot-1|harvest|2026-10-09"))
		second := helper.SHA256Hex([]byte("lot-1|harvest|2026-10-08"))
		if first == second {
			t.Error("expected different hashes for different data")
		}
	})

	t.Run("hash is 64 hex characters", func(t *testing.T) {
		if got := helper.SHA256Hex(nil); len(got) != 64 {
			t.Errorf("got length %d, want 64", len(got))
		}
	})
}
