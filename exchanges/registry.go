package exchanges

import (
	"fmt"
	"log/slog"

	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchange"
	"github.com/pulsoats/core/exchanges/bybit"
)

var ErrExchangeNotFound = fmt.Errorf("exchange %w", errorsx.ErrNotFound)

type Registry struct {
	factories map[string]exchange.EnvFactory
	logger    *slog.Logger
}

// NewRegistry создаёт реестр бирж. Если logger не передан, используется slog.Default().
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	r := &Registry{
		factories: make(map[string]exchange.EnvFactory),
		logger:    logger.With("component", "exchange.registry"),
	}
	r.Register(bybit.Code, bybit.NewFromEnv)
	return r
}

// Register регистрирует фабрику для указанного кода биржи.
func (r *Registry) Register(code string, factory exchange.EnvFactory) {
	r.factories[code] = factory
}

// NewFromEnv создаёт инстанс биржи по коду, читая учётные данные из переменных окружения.
func (r *Registry) NewFromEnv(code string) (exchange.API, error) {
	factory, ok := r.factories[code]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrExchangeNotFound, code)
	}
	api, err := factory(r.logger)
	if err != nil {
		return nil, fmt.Errorf("exchange %s: %w", code, err)
	}
	r.logger.Debug("exchange created", "code", code)
	return api, nil
}
