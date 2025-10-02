package agent

import (
	"math/rand/v2"
	"runtime"
)

type Collector interface {
	PollOnce()
}

type RuntimeStatsProvider interface {
	ReadMemStats(ms *runtime.MemStats)
}

type StdRuntime struct{}

func (StdRuntime) ReadMemStats(ms *runtime.MemStats) { runtime.ReadMemStats(ms) }

type RuntimeCollector struct {
	store LocalStorage
	rt    RuntimeStatsProvider
	rng   *rand.Rand
}

func NewRuntimeCollector(store LocalStorage, rt RuntimeStatsProvider, rng *rand.Rand) *RuntimeCollector {
	if store == nil || rt == nil || rng == nil {
		panic("NewRuntimeCollector: either store, runtime provider or rng is nil")
	}

	return &RuntimeCollector{
		store: store,
		rt:    rt,
		rng:   rng,
	}
}

func (rc *RuntimeCollector) PollOnce() {
	ms := runtime.MemStats{}
	rc.rt.ReadMemStats(&ms)

	rc.store.UpdateGauge("Alloc", float64(ms.Alloc))
	rc.store.UpdateGauge("BuckHashSys", float64(ms.BuckHashSys))
	rc.store.UpdateGauge("Frees", float64(ms.Frees))
	rc.store.UpdateGauge("GCCPUFraction", ms.GCCPUFraction)
	rc.store.UpdateGauge("GCSys", float64(ms.GCSys))
	rc.store.UpdateGauge("HeapAlloc", float64(ms.HeapAlloc))
	rc.store.UpdateGauge("HeapIdle", float64(ms.HeapIdle))
	rc.store.UpdateGauge("HeapInuse", float64(ms.HeapInuse))
	rc.store.UpdateGauge("HeapObjects", float64(ms.HeapObjects))
	rc.store.UpdateGauge("HeapReleased", float64(ms.HeapReleased))
	rc.store.UpdateGauge("HeapSys", float64(ms.HeapSys))
	rc.store.UpdateGauge("LastGC", float64(ms.LastGC))
	rc.store.UpdateGauge("Lookups", float64(ms.Lookups))
	rc.store.UpdateGauge("MCacheInuse", float64(ms.MCacheInuse))
	rc.store.UpdateGauge("MCacheSys", float64(ms.MCacheSys))
	rc.store.UpdateGauge("MSpanInuse", float64(ms.MSpanInuse))
	rc.store.UpdateGauge("MSpanSys", float64(ms.MSpanSys))
	rc.store.UpdateGauge("Mallocs", float64(ms.Mallocs))
	rc.store.UpdateGauge("NextGC", float64(ms.NextGC))
	rc.store.UpdateGauge("NumForcedGC", float64(ms.NumForcedGC))
	rc.store.UpdateGauge("NumGC", float64(ms.NumGC))
	rc.store.UpdateGauge("OtherSys", float64(ms.OtherSys))
	rc.store.UpdateGauge("PauseTotalNs", float64(ms.PauseTotalNs))
	rc.store.UpdateGauge("StackInuse", float64(ms.StackInuse))
	rc.store.UpdateGauge("StackSys", float64(ms.StackSys))
	rc.store.UpdateGauge("Sys", float64(ms.Sys))
	rc.store.UpdateGauge("TotalAlloc", float64(ms.TotalAlloc))
	rc.store.AddCounter("PollCount", 1)
	rc.store.UpdateGauge("RandomValue", rc.rng.Float64())
}
