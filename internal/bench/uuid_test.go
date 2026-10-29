package bench

import "testing"

func TestNewUUIDsSetsVersionFourAndRFCVariant(t *testing.T) {
	batch := make([][16]byte, 64)
	NewUUIDs(batch)
	for _, uuid := range batch {
		if uuid[6]>>4 != 4 {
			t.Errorf("version nibble = %x, want 4", uuid[6]>>4)
		}
		if uuid[8]>>6 != 0b10 {
			t.Errorf("variant bits = %b, want 10", uuid[8]>>6)
		}
	}
	if batch[0] == batch[1] {
		t.Error("two uuids are equal")
	}
}
