package filter

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/market"
)

// Func проверяет найденный сигнал по lookback-данным. Возвращает false, если сигнал нужно отсеять.
type Func func(detectorWindow, lookBackWindow []market.Candle) (bool, error)

type Meta struct {
	Code        string
	Description string
}

type entry struct {
	filterFunc Func
	meta       Meta
}

// Registry хранит фильтры и информацию о них. Ключ - код фильтра.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]entry
}

func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

// Register регистрирует фильтр и его метаданные в реестре. При передаче nil-реестра возникнет паника.
func Register(r *Registry, meta Meta, indicator Func) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[meta.Code]; exists {
		return fmt.Errorf("filter registry: filterFunc=%s: %w", meta.Code, errorsx.ErrAlreadyExists)
	}
	r.entries[meta.Code] = entry{
		filterFunc: indicator,
		meta:       meta,
	}
	return nil
}

func (r *Registry) New(code string) (Func, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, exists := r.entries[code]; !exists {
		return nil, fmt.Errorf("filter registry: filter=%s: %w", code, errorsx.ErrNotFound)
	}
	return r.entries[code].filterFunc, nil
}

func (r *Registry) Meta(code string) (Meta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, exists := r.entries[code]; !exists {
		return Meta{}, fmt.Errorf("filter registry: filter=%s: %w", code, errorsx.ErrNotFound)
	}
	return r.entries[code].meta, nil
}

func (r *Registry) ListMetas() []Meta {
	r.mu.RLock()
	res := make([]Meta, 0, len(r.entries))
	for _, e := range r.entries {
		res = append(res, e.meta)
	}
	r.mu.RUnlock()

	sort.Slice(res, func(i, j int) bool {
		return res[i].Code < res[j].Code
	})
	return res
}
