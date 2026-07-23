package detector

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/pulsoats/core/errorsx"
)

// Factory — конструктор детектора для добавления в Registry.
type Factory func(label string, opts any) (Detector, error)

type entry struct {
	meta    Meta
	optType reflect.Type
	factory Factory
}

func registryKey(code, version string) string {
	return code + "@" + version
}

// Registry хранит фабрики детекторов. Ключ — Code+"@"+Version, позволяя регистрировать несколько версий одного детектора.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]entry
}

func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

// Register добавляет детектор в реестр. Meta.Code и Meta.Version обязательны.
func Register[Opts any](r *Registry, meta Meta, factory func(label string, opts Opts) (Detector, error)) error {
	if meta.Code == "" {
		return fmt.Errorf("detectors registry: detector code: %w", errorsx.ErrRequired)
	}
	if meta.Version == "" {
		return fmt.Errorf("detectors registry: detector=%s version: %w", meta.Code, errorsx.ErrRequired)
	}
	if factory == nil {
		return fmt.Errorf("detectors registry: detector=%s version=%s factory: %w", meta.Code, meta.Version, errorsx.ErrRequired)
	}

	var zero Opts
	t := reflect.TypeOf(zero)

	key := registryKey(meta.Code, meta.Version)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[key]; exists {
		return fmt.Errorf("detectors registry: detector=%s version=%s: %w", meta.Code, meta.Version, errorsx.ErrAlreadyExists)
	}

	r.entries[key] = entry{
		meta:    meta,
		optType: t,
		factory: func(label string, opts any) (Detector, error) {
			o, ok := opts.(Opts)
			if !ok {
				return nil, fmt.Errorf("detectors registry: detector=%s version=%s got=%T want=%s: %w",
					meta.Code, meta.Version, opts, t.String(), errorsx.ErrInvalidArgument,
				)
			}
			det, err := factory(label, o)
			if err != nil {
				return nil, errors.Join(
					fmt.Errorf("detectors registry: detector=%s version=%s factory failed: %w", meta.Code, meta.Version, errorsx.ErrInternal),
					err,
				)
			}
			return det, nil
		},
	}
	return nil
}

// NewFromConfig создает детектор из Config.
func (r *Registry) NewFromConfig(cfg Config) (Detector, error) {
	const op = "detectors registry: new from config"

	key := registryKey(cfg.Code, cfg.Version)

	r.mu.RLock()
	e, ok := r.entries[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%s: detector=%s version=%s: %w", op, cfg.Code, cfg.Version, errorsx.ErrNotFound)
	}

	ptr := reflect.New(e.optType).Interface()
	if err := json.Unmarshal(cfg.Opts, ptr); err != nil {
		return nil, errors.Join(
			fmt.Errorf("%s: detector=%s version=%s unmarshal opts failed: %w", op, cfg.Code, cfg.Version, errorsx.ErrInvalidArgument),
			err,
		)
	}
	opts := reflect.ValueOf(ptr).Elem().Interface()

	result, err := e.factory(cfg.OptsLabel, opts)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("%s: detector=%s version=%s factory failed: %w", op, cfg.Code, cfg.Version, err),
			err,
		)
	}
	return result, nil
}

func (r *Registry) ListMetas() []Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Meta, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.meta)
	}
	return out
}

func (r *Registry) ListVersions(code string) []Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Meta, 0)
	for _, e := range r.entries {
		if e.meta.Code == code {
			out = append(out, e.meta)
		}
	}
	return out
}
