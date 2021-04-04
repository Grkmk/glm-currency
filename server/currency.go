package server

import (
	"context"

	"github.com/grkmk/glm-currency/data"
	"github.com/grkmk/glm-currency/protos/currency"
	"github.com/hashicorp/go-hclog"
)

type Currency struct {
	currency.UnimplementedCurrencyServer
	rates *data.ExchangeRates
	log   hclog.Logger
}

func NewCurrency(r *data.ExchangeRates, l hclog.Logger) *Currency {
	return &Currency{rates: r, log: l}
}

func (c *Currency) GetRate(ctx context.Context, rr *currency.RateRequest) (*currency.RateResponse, error) {
	c.log.Info("handle getrate", "base", rr.GetBase(), "destination", rr.GetDestination())

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &currency.RateResponse{Rate: rate}, nil
}
