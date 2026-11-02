package storage

import (
	"context"
	"testing"
)

func TestMemoryQuarantineGet(t *testing.T) {
	quarantine := NewMemoryQuarantine()
	quarantine.Seed("lot/evidence", []byte("0123456789"))

	missing, err := quarantine.Get(context.Background(), "other", 100)
	if err != nil || missing.Exists {
		t.Errorf("missing object: %+v %v", missing, err)
	}

	whole, _ := quarantine.Get(context.Background(), "lot/evidence", 100)
	if !whole.Exists || len(whole.Data) != 10 {
		t.Errorf("whole object: %+v", whole)
	}

	capped, _ := quarantine.Get(context.Background(), "lot/evidence", 4)
	if len(capped.Data) != 4 {
		t.Errorf("want read capped at 4 bytes, got %d", len(capped.Data))
	}
}

func TestMemoryQuarantineRemove(t *testing.T) {
	quarantine := NewMemoryQuarantine()
	quarantine.Seed("lot/evidence", []byte("x"))

	if err := quarantine.Remove(context.Background(), "lot/evidence"); err != nil {
		t.Fatal(err)
	}

	if object, _ := quarantine.Get(context.Background(), "lot/evidence", 10); object.Exists {
		t.Error("object must be gone")
	}
}

func TestMemoryEvidencesPut(t *testing.T) {
	evidences := NewMemoryEvidences()

	if err := evidences.Put(context.Background(), "lot/evidence/imagen.jpg", "image/jpeg", []byte("x")); err != nil {
		t.Fatal(err)
	}

	keys := evidences.Keys()
	if len(keys) != 1 || keys[0] != "lot/evidence/imagen.jpg" {
		t.Errorf("unexpected keys %v", keys)
	}
}
