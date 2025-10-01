package agent

import (
	"math"
	"math/rand/v2"
	"runtime"
	"testing"

	"github.com/vvolodya/go-metrics/internal/repository"
)

type fakeStore struct{}

func (fakeStore) UpdateGauge(name string, v float64)        {}
func (fakeStore) AddCounter(name string, delta int64) int64 { return 0 }
func (fakeStore) Snapshot() (map[string]float64, map[string]int64) {
	return nil, nil
}

type fakeRT struct{ ms runtime.MemStats }

func (f fakeRT) ReadMemStats(dst *runtime.MemStats) { *dst = f.ms }

func TestNewRuntimeCollector(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))

	tests := []struct {
		name      string
		store     LocalStorage
		rt        RuntimeStatsProvider
		rng       *rand.Rand
		wantPanic bool
	}{
		{
			name:      "all dependencies provided",
			store:     fakeStore{},
			rt:        fakeRT{},
			rng:       rng,
			wantPanic: false,
		},
		{
			name:      "nil store",
			store:     nil,
			rt:        fakeRT{},
			rng:       rng,
			wantPanic: true,
		},
		{
			name:      "nil runtime provider",
			store:     fakeStore{},
			rt:        nil,
			rng:       rng,
			wantPanic: true,
		},
		{
			name:      "nil rng",
			store:     fakeStore{},
			rt:        fakeRT{},
			rng:       nil,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("panic = %v, want %v", r != nil, tt.wantPanic)
				}
			}()

			got := NewRuntimeCollector(tt.store, tt.rt, tt.rng)

			if !tt.wantPanic && got == nil {
				t.Errorf("got nil collector when panic was not expected")
			}
		})
	}
}

func TestRuntimeCollector_PollOnce(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	expRNG := rand.New(rand.NewPCG(1, 2))
	wantRandom := expRNG.Float64()

	ms := runtime.MemStats{
		Alloc:         123,
		BuckHashSys:   234,
		Frees:         345,
		GCCPUFraction: 0.42,
		GCSys:         456,
		HeapAlloc:     567,
		HeapIdle:      678,
		HeapInuse:     789,
		HeapObjects:   890,
		HeapReleased:  901,
		HeapSys:       1011,
		LastGC:        1112,
		Lookups:       1213,
		MCacheInuse:   1314,
		MCacheSys:     1415,
		MSpanInuse:    1516,
		MSpanSys:      1617,
		Mallocs:       1718,
		NextGC:        1819,
		NumForcedGC:   1920,
		NumGC:         21,
		OtherSys:      2223,
		PauseTotalNs:  2324,
		StackInuse:    2425,
		StackSys:      2526,
		Sys:           2627,
		TotalAlloc:    2728,
	}

	store := repository.NewMemStorage()
	rc := NewRuntimeCollector(store, fakeRT{ms: ms}, rng)

	rc.PollOnce()

	checkGauge := func(name string, want float64) {
		if got, ok := store.GetGauge(name); !ok || math.Abs(got-want) > 1e-9 {
			t.Fatalf("gauge %s: want %v, got (%v, ok=%v)", name, want, got, ok)
		}
	}

	checkGauge("Alloc", float64(ms.Alloc))
	checkGauge("BuckHashSys", float64(ms.BuckHashSys))
	checkGauge("Frees", float64(ms.Frees))
	checkGauge("GCCPUFraction", ms.GCCPUFraction)
	checkGauge("GCSys", float64(ms.GCSys))
	checkGauge("HeapAlloc", float64(ms.HeapAlloc))
	checkGauge("HeapIdle", float64(ms.HeapIdle))
	checkGauge("HeapInuse", float64(ms.HeapInuse))
	checkGauge("HeapObjects", float64(ms.HeapObjects))
	checkGauge("HeapReleased", float64(ms.HeapReleased))
	checkGauge("HeapSys", float64(ms.HeapSys))
	checkGauge("LastGC", float64(ms.LastGC))
	checkGauge("Lookups", float64(ms.Lookups))
	checkGauge("MCacheInuse", float64(ms.MCacheInuse))
	checkGauge("MCacheSys", float64(ms.MCacheSys))
	checkGauge("MSpanInuse", float64(ms.MSpanInuse))
	checkGauge("MSpanSys", float64(ms.MSpanSys))
	checkGauge("Mallocs", float64(ms.Mallocs))
	checkGauge("NextGC", float64(ms.NextGC))
	checkGauge("NumForcedGC", float64(ms.NumForcedGC))
	checkGauge("NumGC", float64(ms.NumGC))
	checkGauge("OtherSys", float64(ms.OtherSys))
	checkGauge("PauseTotalNs", float64(ms.PauseTotalNs))
	checkGauge("StackInuse", float64(ms.StackInuse))
	checkGauge("StackSys", float64(ms.StackSys))
	checkGauge("Sys", float64(ms.Sys))
	checkGauge("TotalAlloc", float64(ms.TotalAlloc))

	if got, ok := store.GetCounter("PollCount"); !ok || got != 1 {
		t.Fatalf("counter PollCount: want (1, true), got (%d, %v)", got, ok)
	}

	if rv, ok := store.GetGauge("RandomValue"); !ok || math.Abs(rv-wantRandom) > 1e-12 {
		t.Fatalf("RandomValue: want %v, got (%v, %v)", wantRandom, rv, ok)
	}
}
