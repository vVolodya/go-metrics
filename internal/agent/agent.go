package agent

type LocalStorage interface {
	UpdateGauge(name string, v float64)
	AddCounter(name string, delta int64) int64
	Snapshot() (gauges map[string]float64, counters map[string]int64)
}
