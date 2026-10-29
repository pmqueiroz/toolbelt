package bench

import (
	"context"
	"testing"
	"time"
)

func TestStartGeneratesUntilTheDurationEnds(t *testing.T) {
	run := &Run{Duration: 50 * time.Millisecond, Workers: 2, Bucket: NewBucket(2)}
	result := run.Start(context.Background())

	if result.Generated == 0 {
		t.Fatal("no uuids generated")
	}
	if result.Found > result.Generated {
		t.Errorf("found %d of %d generated", result.Found, result.Generated)
	}
	if result.Generated-result.Found != run.Bucket.Set() {
		t.Errorf("new indexes %d, bucket has %d set", result.Generated-result.Found, run.Bucket.Set())
	}
	if result.Elapsed < run.Duration {
		t.Errorf("elapsed %s, want at least %s", result.Elapsed, run.Duration)
	}
}

func TestStartStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	run := &Run{Duration: time.Hour, Workers: 1, Bucket: NewBucket(1)}
	if result := run.Start(ctx); result.Elapsed > time.Second {
		t.Errorf("cancelled run took %s", result.Elapsed)
	}
}

func TestRateIsZeroWithoutElapsedTime(t *testing.T) {
	if rate := (Result{Generated: 10}).Rate(); rate != 0 {
		t.Errorf("Rate() = %f, want 0", rate)
	}
	if rate := (Result{Generated: 10, Elapsed: 2 * time.Second}).Rate(); rate != 5 {
		t.Errorf("Rate() = %f, want 5", rate)
	}
}
