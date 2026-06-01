package exchanges

import (
	"fmt"
	"log/slog"

	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/exchange"
	"github.com/pulsoats/core/exchanges/bybit"
)

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

	r.register(bybit.Code, func(l *slog.Logger, creds *exchange.Credentials) (exchange.Client, error) {
		return bybit.NewClient(l, creds)
	})
	return r
}

// Register регистрирует фабрику для указанного кода биржи.
func (r *Registry) register(code string, factory exchange.Factory) {
	r.factories[code] = factory
}

// New создаёт авторизованный клиент биржи по коду с переданными учётными данными.
func (r *Registry) New(code string, creds *exchange.Credentials) (exchange.Client, error) {
	return r.new(code, creds)
}

// NewPublic создаёт публичный клиент биржи по коду без авторизации.
func (r *Registry) NewPublic(code string) (exchange.PublicClient, error) {
	return r.new(code, nil)
}

func (r *Registry) CreateAllPublic(logger *slog.Logger) (map[string]exchange.PublicClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	out := make(map[string]exchange.PublicClient, len(r.factories))
	for k, f := range r.factories {
		client, err := f(logger, nil)
		if err != nil {
			return nil, fmt.Errorf("create all public: %w", err)
		}
		out[k] = client
	}

	return out, nil
}

func (r *Registry) new(code string, creds *exchange.Credentials) (exchange.Client, error) {
	factory, ok := r.factories[code]
	if !ok {
		return nil, fmt.Errorf("exchange %s: %w", code, errorsx.ErrNotFound)
	}
	client, err := factory(r.logger, creds)
	if err != nil {
		return nil, fmt.Errorf("exchange %s: %w", code, err)
	}
	r.logger.Debug("exchange created", "code", code, "auth", creds != nil)
	return client, nil
}
