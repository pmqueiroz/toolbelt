package bench

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const batchSize = 1024

type Result struct {
	Generated uint64
	Found     uint64
	Elapsed   time.Duration
	Workers   int
}

func (r Result) Rate() float64 {
	if r.Elapsed <= 0 {
		return 0
	}
	return float64(r.Generated) / r.Elapsed.Seconds()
}

type Run struct {
	Duration  time.Duration
	Workers   int
	Bucket    *Bucket
	Generated atomic.Uint64
	found     atomic.Uint64
}

func (r *Run) Start(ctx context.Context) Result {
	ctx, cancel := context.WithTimeout(ctx, r.Duration)
	defer cancel()

	started := time.Now()
	var wg sync.WaitGroup
	for range r.Workers {
		wg.Go(func() { r.work(ctx) })
	}
	wg.Wait()

	return Result{
		Generated: r.Generated.Load(),
		Found:     r.found.Load(),
		Elapsed:   time.Since(started),
		Workers:   r.Workers,
	}
}

func (r *Run) work(ctx context.Context) {
	batch := make([][16]byte, batchSize)
	var key [encodedLength]byte
	for ctx.Err() == nil {
		NewUUIDs(batch)
		var found uint64
		for i := range batch {
			if r.Bucket.Mark(Key(&key, &batch[i])) {
				found++
			}
		}
		r.Generated.Add(batchSize)
		r.found.Add(found)
	}
}
