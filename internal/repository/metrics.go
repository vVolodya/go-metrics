package repository

import "sync"

type MetricsStorage interface {
	UpdateGauge(name string, v float64)
	AddCounter(name string, delta int64) int64
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	//Snapshot() (map[string]float64, map[string]int64)
}

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
	rwMutex sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(name string, v float64) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	m.gauge[name] = v
}

func (m *MemStorage) AddCounter(name string, delta int64) int64 {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	prev := m.counter[name]
	m.counter[name] = prev + delta
	return m.counter[name]
}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	m.rwMutex.RLock()
	v, exists := m.gauge[name]
	m.rwMutex.RUnlock()

	return v, exists
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	m.rwMutex.RLock()
	v, exists := m.counter[name]
	m.rwMutex.RUnlock()

	return v, exists
}

// Контракт на реализацию
// Это приём, когда в коде ты явно заставляешь компилятор проверить, что структура действительно реализует интерфейс. Делается через пустое присваивание:
// (*MemStorage)(nil) — это указатель на MemStorage (значение nil).
// var _ MetricsStorage = ... — компилятор должен проверить: можно ли *MemStorage присвоить переменной типа MetricsStorage?
// Если да → всё ок.
// Если нет (например, ты удалил метод или поменял сигнатуру) → ошибка компиляции.
var _ MetricsStorage = (*MemStorage)(nil)
