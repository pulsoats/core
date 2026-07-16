package detector

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
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

func (r *Registry) New(code, version, label string, opts any) (Detector, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	e, ok := r.entries[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	result, err := e.factory(label, opts)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s version=%s factory failed: %w", code, version, errorsx.ErrInternal),
			err,
		)
	}
	return result, nil
}

func (r *Registry) Meta(code, version string) (Meta, bool) {
	key := registryKey(code, version)

	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.entries[key]
	return e.meta, ok
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

	out := make([]Meta, 0)
	for _, e := range r.entries {
		if e.meta.Code == code {
			out = append(out, e.meta)
		}
	}
	r.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Code < out[j].Code
	})
	return out
}

func (r *Registry) NewOptsPtr(code, version string) (any, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	e, ok := r.entries[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	return reflect.New(e.optType).Interface(), nil
}

func (r *Registry) MarshalOpts(code, version string, opts any) ([]byte, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	e, ok := r.entries[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	if opts == nil {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s got=<nil> want=%s: %w",
			code, version, e.optType.String(), errorsx.ErrInvalidArgument)
	}

	gotType := reflect.TypeOf(opts)
	var (
		data []byte
		err  error
	)
	switch {
	case gotType == e.optType:
		data, err = json.Marshal(opts)
	case gotType.Kind() == reflect.Pointer && gotType.Elem() == e.optType:
		data, err = json.Marshal(reflect.ValueOf(opts).Elem().Interface())
	default:
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s got=%s want=%s: %w",
			code, version, gotType.String(), e.optType.String(), errorsx.ErrInvalidArgument)
	}
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s version=%s marshal opts failed: %w", code, version, errorsx.ErrInvalidArgument),
			err,
		)
	}
	return data, nil
}

func (r *Registry) UnmarshalOpts(code, version string, data []byte) (any, error) {
	ptr, err := r.NewOptsPtr(code, version)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, ptr); err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s version=%s unmarshal opts failed: %w", code, version, errorsx.ErrInvalidArgument),
			err,
		)
	}

	return reflect.ValueOf(ptr).Elem().Interface(), nil
}
