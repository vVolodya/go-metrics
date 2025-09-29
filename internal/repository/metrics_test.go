package repository

import (
	"sync"
	"testing"
)

func TestAddCounter_SumsValues(t *testing.T) {
	s := NewMemStorage()

	if got := s.AddCounter("x", 10); got != 10 {
		t.Fatalf("first add: want 10, got %d", got)
	}
	if got := s.AddCounter("x", 5); got != 15 {
		t.Fatalf("second add: want 15, got %d", got)
	}
	if v, ok := s.GetCounter("x"); !ok || v != 15 {
		t.Fatalf("GetCounter: want (15, true), got (%d, %v)", v, ok)
	}
}

func TestUpdateGauge_Overwrites(t *testing.T) {
	s := NewMemStorage()

	s.UpdateGauge("t", 1.5)
	if v, ok := s.GetGauge("t"); !ok || v != 1.5 {
		t.Fatalf("after first update: want (1.5, true), got (%v, %v)", v, ok)
	}

	s.UpdateGauge("t", 2.0)
	if v, ok := s.GetGauge("t"); !ok || v != 2.0 {
		t.Fatalf("after second update: want (2.0, true), got (%v, %v)", v, ok)
	}
}

func TestAddCounter_Concurrent(t *testing.T) {
	s := NewMemStorage()

	const workers = 8
	const iters = 10_000

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				s.AddCounter("c", 1)
			}
		}()
	}

	wg.Wait()

	want := int64(workers * iters)
	if got, ok := s.GetCounter("c"); !ok || got != want {
		t.Fatalf("concurrent sum: want (%d, true), got (%d, %v)", want, got, ok)
	}
}
