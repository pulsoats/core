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
	factories map[string]exchange.Factory
	logger    *slog.Logger
}

// NewRegistry создаёт реестр бирж. Если logger не передан, используется slog.Default().
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}

	r := &Registry{
		factories: make(map[string]exchange.Factory),
		logger:    logger.With("component", "exchange.registry"),
	}

	r.register(bybit.Code, func(l *slog.Logger, auth bool) (exchange.Client, error) {
		return bybit.NewClient(l, auth)
	})
	return r
}

// Register регистрирует фабрику для указанного кода биржи.
func (r *Registry) register(code string, factory exchange.Factory) {
	r.factories[code] = factory
}

// NewFromEnv создаёт клиент биржи по коду с авторизацией, читая учётные данные из переменных окружения.
func (r *Registry) NewFromEnv(code string) (exchange.Client, error) {
	return r.new(code, true)
}

// NewPublic создаёт публичный клиент биржи по коду без авторизации.
func (r *Registry) NewPublic(code string) (exchange.Client, error) {
	return r.new(code, false)
}

func (r *Registry) CreateAllPublic(logger *slog.Logger) (map[string]exchange.Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	out := make(map[string]exchange.Client, len(r.factories))
	for k, f := range r.factories {
		client, err := f(logger, false)
		if err != nil {
			return nil, fmt.Errorf("create all public: %w", err)
		}
		out[k] = client
	}

	return out, nil
}
func (r *Registry) new(code string, auth bool) (exchange.Client, error) {
	factory, ok := r.factories[code]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrExchangeNotFound, code)
	}
	client, err := factory(r.logger, auth)
	if err != nil {
		return nil, fmt.Errorf("exchange %s: %w", code, err)
	}
	r.logger.Debug("exchange created", "code", code, "auth", auth)
	return client, nil
}
