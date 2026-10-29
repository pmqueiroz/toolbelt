package bench

import "testing"

func TestReadCapacityReportsMemoryAndDisk(t *testing.T) {
	capacity, err := ReadCapacity(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if capacity.Memory == 0 || capacity.Disk == 0 {
		t.Errorf("capacity = %+v, want both non-zero", capacity)
	}
}
