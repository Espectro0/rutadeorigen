package lot

import "github.com/google/uuid"

func VerifyChain(lotID uuid.UUID, events []*Event) bool {
	previousHash := GenesisHash
	for i, e := range events {
		if e.IsDefault() || e.lotID != lotID {
			return false
		}
		if e.sequence != firstSequence+i || e.previousHash != previousHash || !e.VerifyHash() {
			return false
		}
		previousHash = e.hash
	}
	return true
}
