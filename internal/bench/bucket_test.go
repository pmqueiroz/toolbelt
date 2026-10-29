package bench

import "testing"

func TestIndexDecodesBase64PrefixIntoBits(t *testing.T) {
	cases := []struct {
		key   string
		chars int
		want  uint64
	}{
		{"AAAAAAAA", 5, 0},
		{"BAAAAAAA", 2, 64},
		{"AB", 2, 1},
		{"_____xyz", 5, 1<<30 - 1},
	}
	for _, c := range cases {
		if got := Index([]byte(c.key), c.chars); got != c.want {
			t.Errorf("Index(%q, %d) = %d, want %d", c.key, c.chars, got, c.want)
		}
	}
}

func TestMarkReportsWhetherTheIndexWasAlreadyFound(t *testing.T) {
	bucket := NewBucket(2)
	if bucket.Mark([]byte("AB")) {
		t.Error("first mark reported found")
	}
	if !bucket.Mark([]byte("ABzz")) {
		t.Error("second mark on same index reported not found")
	}
	if bucket.Mark([]byte("BA")) {
		t.Error("mark on another index reported found")
	}
	if got := bucket.Set(); got != 2 {
		t.Errorf("Set() = %d, want 2", got)
	}
}

func TestBucketHoldsOneBitPerIndex(t *testing.T) {
	bucket := NewBucket(IndexChars)
	if bucket.Indexes() != 1<<30 {
		t.Errorf("Indexes() = %d, want %d", bucket.Indexes(), 1<<30)
	}
	if bucket.Bytes() != 1<<27 {
		t.Errorf("Bytes() = %d, want %d", bucket.Bytes(), 1<<27)
	}
}

func TestKeyIsUnpaddedBase64OfTheUUID(t *testing.T) {
	uuid := [16]byte{0xff, 0xff, 0xff, 0xff}
	var dst [encodedLength]byte
	if got := string(Key(&dst, &uuid)); got != "_____wAAAAAAAAAAAAAAAA" {
		t.Errorf("Key() = %q", got)
	}
}
