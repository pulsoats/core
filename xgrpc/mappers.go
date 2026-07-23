package xgrpc

import (
	"fmt"

	corepb "github.com/pulsoats/contracts/gen/go/core/v1"
	"github.com/pulsoats/core/detect/detector"
	"github.com/pulsoats/core/detect/filter"
	"github.com/pulsoats/core/errorsx"
	"github.com/pulsoats/core/market"
)

func MarketSpecToProto(spec market.Spec) *corepb.MarketSpec {
	return &corepb.MarketSpec{
		Exchange: spec.Exchange,
		Category: spec.Category,
		Symbol:   spec.Symbol,
	}
}

func MarketSpecFromProto(pb *corepb.MarketSpec) (market.Spec, error) {
	if pb == nil {
		return market.Spec{}, fmt.Errorf("market spec from proto: message is nil: %w", errorsx.ErrInvalidArgument)
	}
	return market.Spec{
		Exchange: pb.Exchange,
		Category: pb.Category,
		Symbol:   pb.Symbol,
	}, nil
}

func DetectorConfigToProto(cfg detector.Config) *corepb.DetectorConfig {
	return &corepb.DetectorConfig{
		Code:      cfg.Code,
		Version:   cfg.Version,
		OptsLabel: cfg.OptsLabel,
		Opts:      cfg.Opts,
	}
}

func DetectorConfigFromProto(pb *corepb.DetectorConfig) (detector.Config, error) {
	if pb == nil {
		return detector.Config{}, fmt.Errorf("detector config from proto: message is nil: %w", errorsx.ErrInvalidArgument)
	}
	return detector.Config{
		Code:      pb.Code,
		Version:   pb.Version,
		OptsLabel: pb.OptsLabel,
		Opts:      pb.Opts,
	}, nil
}

func FilterConfigToProto(cfg filter.Config) *corepb.FilterConfig {
	return &corepb.FilterConfig{
		Code:   cfg.Code,
		Period: int32(cfg.Period),
	}
}

func FilterConfigFromProto(pb *corepb.FilterConfig) (filter.Config, error) {
	if pb == nil {
		return filter.Config{}, fmt.Errorf("filter config from proto: message is nil: %w", errorsx.ErrInvalidArgument)
	}
	return filter.Config{
		Code:   pb.Code,
		Period: int(pb.Period),
	}, nil
}

func FeesToProto(fees *market.TakerMakerFees) *corepb.Fees {
	if fees == nil {
		return nil
	}
	return &corepb.Fees{
		TakerFeePpm: fees.TakerFeeRate,
		MakerFeePpm: fees.MakerFeeRate,
	}
}

func FeesFromProto(pb *corepb.Fees) *market.TakerMakerFees {
	if pb == nil {
		return nil
	}
	return &market.TakerMakerFees{
		TakerFeeRate: pb.TakerFeePpm,
		MakerFeeRate: pb.MakerFeePpm,
	}
}
