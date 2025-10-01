package agent

import "github.com/vvolodya/go-metrics/internal/repository"

var _ LocalStorage = (*repository.MemStorage)(nil)
