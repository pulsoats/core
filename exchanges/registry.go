package exchanges

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/pulsoats/core/domain/derrors"
	"github.com/pulsoats/core/domain/exchange"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/exchanges/bybit"
	"github.com/pulsoats/core/lib/logx"
)

type Registry struct {
	mu        sync.RWMutex
	factories map[string]exchange.Factory
	meta      map[string]exchange.Meta
	logger    logx.Logger
}

type Option func(registry *Registry)

func WithLogger(logger logx.Logger) Option {
	return func(r *Registry) {
		if logger == nil {
			logger = logx.Nop()
		}
		r.logger = logger
	}
}

func NewRegistry(opts ...Option) *Registry {
	r := &Registry{
		factories: make(map[string]exchange.Factory),
		meta:      make(map[string]exchange.Meta),
		logger:    logx.Nop(),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.mustAdd(bybit.Metadata, func(apiKey, apiSecret string) (exchange.API, error) {
		return bybit.NewBybitClient(apiKey, apiSecret, bybit.WithLogger(r.logger)), nil
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
		return fmt.Errorf("%w: exchange=%s", derrors.ErrAlreadyExists, meta.Code)
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
		return nil, errors.Join(exchange.ErrFactoryFailed, err)
	}
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
	if len(meta.PriceTypes) > 0 {
		cloned.PriceTypes = append([]market.PriceType(nil), meta.PriceTypes...)
	} else {
		cloned.PriceTypes = nil
	}
	return cloned
}
