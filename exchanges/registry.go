package exchanges

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/pulsoats/core/domain/exchange"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchanges/bybit"
)

type Registry struct {
	mu        sync.RWMutex
	factories map[string]exchange.Factory
	meta      map[string]exchange.Meta
	logger    *slog.Logger
}

type Option func(registry *Registry)

func WithLogger(logger *slog.Logger) Option {
	return func(r *Registry) {
		if logger == nil {
			logger = slog.New(slog.DiscardHandler)
		}
		r.logger = logger
	}
}

func NewRegistry(opts ...Option) *Registry {
	r := &Registry{
		factories: make(map[string]exchange.Factory),
		meta:      make(map[string]exchange.Meta),
		logger:    slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(r)
	}
	childLogger := r.logger // children label themselves independently
	r.logger = r.logger.With("component", "exchange.registry")
	r.mustAdd(bybit.Metadata, func(apiKey, apiSecret string) (exchange.API, error) {
		return bybit.NewBybitClient(apiKey, apiSecret, bybit.WithLogger(childLogger)), nil
	})
	return r
}

func (r *Registry) add(meta exchange.Meta, factory exchange.Factory) error {
	if meta.Code == "" {
		return exchange.ErrExchangeEmpty
	}
	if factory == nil {
		return exchange.ErrFactoryNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[meta.Code]; exists {
		return fmt.Errorf("exchange registry: exchange=%s: %w", meta.Code, errorsx.ErrAlreadyExists)
	}

	r.factories[meta.Code] = factory
	r.meta[meta.Code] = cloneMetadata(meta)
	return nil
}

func (r *Registry) mustAdd(meta exchange.Meta, factory exchange.Factory) {
	if err := r.add(meta, factory); err != nil {
		panic(err)
	}
}

func (r *Registry) Create(code string, apiKey string, apiSecret string) (exchange.API, error) {
	r.mu.RLock()
	factory, ok := r.factories[code]
	r.mu.RUnlock()
	if !ok {
		return nil, exchange.ErrExchangeNotFound
	}
	api, err := factory(apiKey, apiSecret)
	if err != nil {
		r.logger.Error("exchange factory failed", "code", code, "err", err)
		return nil, errors.Join(exchange.ErrFactoryFailed, err)
	}
	r.logger.Debug("exchange created", "code", code)
	return api, nil
}

func (r *Registry) CreateAllPublic() (map[string]exchange.API, error) {
	r.mu.RLock()
	factories := make(map[string]exchange.Factory, len(r.factories))
	for code, factory := range r.factories {
		factories[code] = factory
	}
	r.mu.RUnlock()

	result := make(map[string]exchange.API, len(factories))
	for code, factory := range factories {
		api, err := factory("", "")
		if err != nil {
			return nil, errors.Join(exchange.ErrFactoryFailed, err)
		}
		result[code] = api
	}
	return result, nil
}

func (r *Registry) Metadata(code string) (exchange.Meta, bool) {
	r.mu.RLock()
	meta, ok := r.meta[code]
	r.mu.RUnlock()
	if !ok {
		return exchange.Meta{}, false
	}
	return cloneMetadata(meta), true
}

func (r *Registry) ListMetadata() []exchange.Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]exchange.Meta, 0, len(r.meta))
	for _, meta := range r.meta {
		out = append(out, cloneMetadata(meta))
	}
	return out
}

// List returns list of exchange codes in ASC order
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.factories))
	for code := range r.factories {
		out = append(out, code)
	}

	slices.Sort(out)
	return out
}

func cloneMetadata(meta exchange.Meta) exchange.Meta {
	cloned := meta
	if len(meta.Intervals) > 0 {
		cloned.Intervals = append([]market.Interval(nil), meta.Intervals...)
	} else {
		cloned.Intervals = nil
	}
	if len(meta.Categories) > 0 {
		cloned.Categories = append([]market.Category(nil), meta.Categories...)
	} else {
		cloned.Categories = nil
	}
	return cloned
}
