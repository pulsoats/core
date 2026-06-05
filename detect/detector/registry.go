package detector

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/pulsoats/core/detect"
	"github.com/pulsoats/core/errorsx"
)

type Factory func(label string, opts any) (detect.Detector, error)

func registryKey(code, version string) string {
	return code + "@" + version
}

// Registry хранит фабрики и метаданные детекторов любого типа.
// Ключ реестра — составной: Code + "@" + Version, что позволяет
// иметь несколько версий одного детектора одновременно.
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

func (r *Registry) registerMeta(meta detect.DetectorMeta, optType reflect.Type, factory Factory) error {
	if meta.Code == "" {
		return fmt.Errorf("detectors registry: detector code: %w", errorsx.ErrRequired)
	}
	if meta.Version == "" {
		return fmt.Errorf("detectors registry: detector=%s version: %w", meta.Code, errorsx.ErrRequired)
	}
	if optType == nil {
		return fmt.Errorf("detectors registry: detector=%s version=%s opts type: %w", meta.Code, meta.Version, errorsx.ErrRequired)
	}
	if factory == nil {
		return fmt.Errorf("detectors registry: detector=%s version=%s factory: %w", meta.Code, meta.Version, errorsx.ErrRequired)
	}

	key := registryKey(meta.Code, meta.Version)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.meta[key]; exists {
		return fmt.Errorf("detectors registry: detector=%s version=%s: %w", meta.Code, meta.Version, errorsx.ErrAlreadyExists)
	}

	r.meta[key] = meta
	r.optTypes[key] = optType
	r.factories[key] = factory
	return nil
}

func registerDetector[Opts any, Det detect.Detector](
	r *Registry,
	meta detect.DetectorMeta,
	factory func(label string, opts Opts) (Det, error),
) error {
	var zero Opts
	t := reflect.TypeOf(zero)

	return r.registerMeta(meta, t, func(label string, opts any) (detect.Detector, error) {
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
	})
}

// AddCandleDetector регистрирует CandleDetector в реестре.
// Вызывается внешними пакетами (например, github.com/pulsoats/detectors).
// meta.Version обязателен и используется как часть ключа реестра.
func AddCandleDetector[Opts any](
	r *Registry,
	meta detect.DetectorMeta,
	factory func(label string, opts Opts) (detect.CandleDetector, error),
) error {
	if meta.Kind != detect.DetectorKindCandle {
		return fmt.Errorf("detectors registry: detector=%s version=%s: unexpected kind=%s want=%s: %w",
			meta.Code, meta.Version, meta.Kind, detect.DetectorKindCandle, errorsx.ErrInternal)
	}
	return registerDetector(r, meta, factory)
}

func (r *Registry) ensureKind(code, version string, kind detect.DetectorKind) (detect.DetectorMeta, error) {
	meta, ok := r.Get(code, version)
	if !ok || meta.Kind != kind {
		return detect.DetectorMeta{}, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	return meta, nil
}

func (r *Registry) NewCandle(code, version, label string, opts any) (detect.CandleDetector, error) {
	if _, err := r.ensureKind(code, version, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	det, err := r.New(code, version, label, opts)
	if err != nil {
		return nil, err
	}
	cd, ok := det.(detect.CandleDetector)
	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s kind=%s want=%s: %w",
			det.Code(), version, det.Kind(), detect.DetectorKindCandle, errorsx.ErrInternal)
	}
	return cd, nil
}

func (r *Registry) NewCandleOptsPtr(code, version string) (any, error) {
	if _, err := r.ensureKind(code, version, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.NewOptsPtr(code, version)
}

func (r *Registry) GetCandle(code, version string) (detect.DetectorMeta, bool) {
	meta, err := r.ensureKind(code, version, detect.DetectorKindCandle)
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

// ListVersions возвращает все зарегистрированные версии детектора с заданным кодом.
func (r *Registry) ListVersions(code string) []detect.DetectorMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]detect.DetectorMeta, 0)
	for _, m := range r.meta {
		if m.Code == code {
			out = append(out, m)
		}
	}
	return out
}

func (r *Registry) MarshalCandleOpts(code, version string, opts any) ([]byte, error) {
	if _, err := r.ensureKind(code, version, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.MarshalOpts(code, version, opts)
}

func (r *Registry) UnmarshalCandleOpts(code, version string, data []byte) (any, error) {
	if _, err := r.ensureKind(code, version, detect.DetectorKindCandle); err != nil {
		return nil, err
	}
	return r.UnmarshalOpts(code, version, data)
}

func (r *Registry) New(code, version, label string, opts any) (detect.Detector, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	factory, ok := r.factories[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	result, err := factory(label, opts)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("detectors registry: detector=%s version=%s factory failed: %w", code, version, errorsx.ErrInternal),
			err,
		)
	}
	return result, nil
}

func (r *Registry) NewOptsPtr(code, version string) (any, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	optType, ok := r.optTypes[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	return reflect.New(optType).Interface(), nil
}

func (r *Registry) Get(code, version string) (detect.DetectorMeta, bool) {
	key := registryKey(code, version)

	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.meta[key]
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

func (r *Registry) MarshalOpts(code, version string, opts any) ([]byte, error) {
	key := registryKey(code, version)

	r.mu.RLock()
	optType, ok := r.optTypes[key]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s: %w", code, version, errorsx.ErrNotFound)
	}
	if opts == nil {
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s got=<nil> want=%s: %w",
			code, version, optType.String(), errorsx.ErrInvalidArgument)
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
		return nil, fmt.Errorf("detectors registry: detector=%s version=%s got=%s want=%s: %w",
			code, version, gotType.String(), optType.String(), errorsx.ErrInvalidArgument)
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
