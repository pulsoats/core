package detectors

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/domain/detect"
	"github.com/pulsoats/core/lib/errorsx"
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
		return fmt.Errorf("%w: detector meta name is empty", derrors.ErrRequired)
	}
	if optType == nil {
		return fmt.Errorf("%w: detector=%s opts type is nil", derrors.ErrRequired, meta.Code)
	}
	if factory == nil {
		return fmt.Errorf("%w: detector=%s factory is nil", derrors.ErrRequired, meta.Code)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.meta[meta.Code]; exists {
		return fmt.Errorf("%w: detector=%s", derrors.ErrAlreadyExists, meta.Code)
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
			return nil, fmt.Errorf("%w: detector=%s got=%T want=%s",
				derrors.ErrInvalidArgument, code, opts, t.String(),
			)
		}

		det, err := factory(label, o)
		if err != nil {
			return nil, fmt.Errorf("%w: detector=%s factory failed: %w",
				errorsx.ErrInternal, code, err,
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
		return detect.DetectorMeta{}, fmt.Errorf("%w: detector=%s", derrors.ErrNotFound, name)
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
		return nil, fmt.Errorf("%w: detector=%s kind=%s want=%s", errorsx.ErrInternal, det.Code(), det.Kind(), detect.DetectorKindCandle)
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
		return nil, fmt.Errorf("%w: detector=%s", derrors.ErrNotFound, name)
	}
	result, err := factory(label, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: detector=%s factory failed: %w", errorsx.ErrInternal, name, err)
	}
	return result, nil
}

func (r *Registry) NewOptsPtr(name string) (any, error) {
	r.mu.RLock()
	optType, ok := r.optTypes[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: detector=%s", derrors.ErrNotFound, name)
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
		return nil, fmt.Errorf("%w: detector=%s", derrors.ErrNotFound, name)
	}
	if opts == nil {
		return nil, fmt.Errorf("%w: detector=%s got=<nil> want=%s", derrors.ErrInvalidArgument, name, optType.String())
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
		return nil, fmt.Errorf("%w: detector=%s got=%s want=%s", derrors.ErrInvalidArgument, name, gotType.String(), optType.String())
	}
	if err != nil {
		return nil, fmt.Errorf("%w: detector=%s marshal opts failed: %w", derrors.ErrInvalidArgument, name, err)
	}
	return data, nil
}

func (r *Registry) UnmarshalOpts(name string, data []byte) (any, error) {
	ptr, err := r.NewOptsPtr(name)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, ptr); err != nil {
		return nil, fmt.Errorf("%w: detector=%s unmarshal opts failed: %w", derrors.ErrInvalidArgument, name, err)
	}

	return reflect.ValueOf(ptr).Elem().Interface(), nil
}
