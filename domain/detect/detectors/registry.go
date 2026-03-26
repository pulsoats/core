package detectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/pulsoats/core/domain/detect"
	"github.com/pulsoats/core/errorsx"
)

type Factory func(label string, opts any) (detect.Detector, error)

// Registry хранит фабрики и метаданные детекторов любого типа.
type Registry struct {
	mu        sync.RWMutex
	meta      map[string]detect.DetectorMeta
	optTypes  map[string]reflect.Type
	factories map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{
		meta:      make(map[string]detect.DetectorMeta),
		optTypes:  make(map[string]reflect.Type),
		factories: make(map[string]Factory),
	}
}

// RegisterAll добавляет все встроенные детекторы в переданный реестр.
func RegisterAll(r *Registry) error {
	return registerBuiltins(r)
}

// NewDefaultRegistry возвращает реестр, заполненный всеми встроенными детекторами.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	if err := RegisterAll(r); err != nil {
		panic(err)
	}
	return r
}

func registerBuiltins(r *Registry) error {
	return registerCandleDetector(r, "W", WDescription, WOptsSchema, NewWDetector)
}

func (r *Registry) registerMeta(meta detect.DetectorMeta, optType reflect.Type, factory Factory) error {
	if meta.Code == "" {
		return fmt.Errorf("detectors registry: detector meta name: %w", errorsx.ErrRequired)
	}
	if optType == nil {
		return fmt.Errorf("detectors registry: detector=%s opts type: %w", meta.Code, errorsx.ErrRequired)
	}
	if factory == nil {
		return fmt.Errorf("detectors registry: detector=%s factory: %w", meta.Code, errorsx.ErrRequired)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.meta[meta.Code]; exists {
		return fmt.Errorf("detectors registry: detector=%s: %w", meta.Code, errorsx.ErrAlreadyExists)
	}

	r.meta[meta.Code] = meta
	r.optTypes[meta.Code] = optType
	r.factories[meta.Code] = factory
	return nil
}

func registerDetector[Opts any, Det detect.Detector](
	r *Registry,
	kind detect.DetectorKind,
	code string,
	desc string,
	optsSchema json.RawMessage,
	factory func(label string, opts Opts) (Det, error),
) error {
	var zero Opts
	t := reflect.TypeOf(zero)

	meta := detect.DetectorMeta{
		Code:       code,
		Desc:       desc,
		OptsSchema: optsSchema,
		Kind:       kind,
	}

	return r.registerMeta(meta, t, func(label string, opts any) (detect.Detector, error) {
		o, ok := opts.(Opts)
		if !ok {
			return nil, fmt.Errorf("detectors registry: detector=%s got=%T want=%s: %w",
				code, opts, t.String(), errorsx.ErrInvalidArgument,
			)
		}

		det, err := factory(label, o)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("detectors registry: detector=%s factory failed: %w", code, errorsx.ErrInternal),
				err,
			)
		}
		return det, nil
	})
}

func registerCandleDetector[Opts any](
	r *Registry,
	code, desc string,
	optsSchema json.RawMessage,
	factory func(label string, opts Opts) (detect.CandleDetector, error),
) error {
	return registerDetector(r, detect.DetectorKindCandle, code, desc, optsSchema, factory)
}

func (r *Registry) ensureKind(name string, kind detect.DetectorKind) (detect.DetectorMeta, error) {
	meta, ok := r.Get(name)
	if !ok || meta.Kind != kind {
		return detect.DetectorMeta{}, fmt.Errorf("detectors registry: detector=%s: %w", name, errorsx.ErrNotFound)
	}
	return meta, nil
}

func (r *Registry) NewCandle(name, label string, opts any) (detect.CandleDetector, error) {
	if _, err := r.ensureKind(name, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	det, err := r.New(name, label, opts)
	if err != nil {
		return nil, err
	}
	cd, ok := det.(detect.CandleDetector)
	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s kind=%s want=%s: %w", det.Code(), det.Kind(), detect.DetectorKindCandle, errorsx.ErrInternal)
	}
	return cd, nil
}

func (r *Registry) NewCandleOptsPtr(name string) (any, error) {
	if _, err := r.ensureKind(name, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.NewOptsPtr(name)
}

func (r *Registry) GetCandle(name string) (detect.DetectorMeta, bool) {
	meta, err := r.ensureKind(name, detect.DetectorKindCandle)
	if err != nil {
		return detect.DetectorMeta{}, false
	}
	return meta, true
}

func (r *Registry) ListCandleMetas() []detect.DetectorMeta {
	all := r.ListMetas()
	out := make([]detect.DetectorMeta, 0, len(all))
	for _, meta := range all {
		if meta.Kind == detect.DetectorKindCandle {
			out = append(out, meta)
		}
	}
	return out
}

func (r *Registry) MarshalCandleOpts(name string, opts any) ([]byte, error) {
	if _, err := r.ensureKind(name, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.MarshalOpts(name, opts)
}

func (r *Registry) UnmarshalCandleOpts(name string, data []byte) (any, error) {
	if _, err := r.ensureKind(name, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.UnmarshalOpts(name, data)
}

func (r *Registry) New(name string, label string, opts any) (detect.Detector, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s: %w", name, errorsx.ErrNotFound)
	}
	result, err := factory(label, opts)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s factory failed: %w", name, errorsx.ErrInternal),
			err,
		)
	}
	return result, nil
}

func (r *Registry) NewOptsPtr(name string) (any, error) {
	r.mu.RLock()
	optType, ok := r.optTypes[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s: %w", name, errorsx.ErrNotFound)
	}
	return reflect.New(optType).Interface(), nil
}

func (r *Registry) Get(name string) (detect.DetectorMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.meta[name]
	return m, ok
}

func (r *Registry) ListMetas() []detect.DetectorMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]detect.DetectorMeta, 0, len(r.meta))
	for _, m := range r.meta {
		out = append(out, m)
	}
	return out
}

func (r *Registry) MarshalOpts(name string, opts any) ([]byte, error) {
	r.mu.RLock()
	optType, ok := r.optTypes[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s: %w", name, errorsx.ErrNotFound)
	}
	if opts == nil {
		return nil, fmt.Errorf("detectors registry: detector=%s got=<nil> want=%s: %w", name, optType.String(), errorsx.ErrInvalidArgument)
	}
	gotType := reflect.TypeOf(opts)
	var (
		data []byte
		err  error
	)
	switch {
	case gotType == optType:
		data, err = json.Marshal(opts)
	case gotType.Kind() == reflect.Pointer && gotType.Elem() == optType:
		data, err = json.Marshal(reflect.ValueOf(opts).Elem().Interface())
	default:
		return nil, fmt.Errorf("detectors registry: detector=%s got=%s want=%s: %w", name, gotType.String(), optType.String(), errorsx.ErrInvalidArgument)
	}
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s marshal opts failed: %w", name, errorsx.ErrInvalidArgument),
			err,
		)
	}
	return data, nil
}

func (r *Registry) UnmarshalOpts(name string, data []byte) (any, error) {
	ptr, err := r.NewOptsPtr(name)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, ptr); err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s unmarshal opts failed: %w", name, errorsx.ErrInvalidArgument),
			err,
		)
	}

	return reflect.ValueOf(ptr).Elem().Interface(), nil
}
