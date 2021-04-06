package server

import (
	"context"
	"io"
	"time"

	"github.com/grkmk/glm-currency/data"
	"github.com/grkmk/glm-currency/protos/currency"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Currency struct {
	currency.UnimplementedCurrencyServer
	rates         *data.ExchangeRates
	log           hclog.Logger
	subscriptions map[currency.Currency_SubscribeRatesServer][]*currency.RateRequest
}

func NewCurrency(r *data.ExchangeRates, l hclog.Logger) *Currency {
	c := &Currency{
		rates:         r,
		log:           l,
		subscriptions: make(map[currency.Currency_SubscribeRatesServer][]*currency.RateRequest),
	}

	go c.handleUpdates()

	return c
}

func (c *Currency) handleUpdates() {
	ru := c.rates.MonitorRates(5 * time.Second)
	for range ru {
		c.log.Info("Got updated rates")

		for k, v := range c.subscriptions {

			for _, rr := range v {
				r, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
				if err != nil {
					c.log.Error("Unable to get updated rate", "base", rr.GetBase().String(), "destination", rr.GetDestination())
				}

				err = k.Send(
					&currency.StreamingRateResponse{
						Message: &currency.StreamingRateResponse_RateResponse{
							RateResponse: &currency.RateResponse{
								Base:        rr.Base,
								Destination: rr.Destination,
								Rate:        r,
							},
						},
					},
				)
				if err != nil {
					c.log.Error("Unable to send updated rate", "base", rr.GetBase().String(), "destination", rr.GetDestination())
				}
			}

		}
	}
}

func (c *Currency) GetRate(ctx context.Context, rr *currency.RateRequest) (*currency.RateResponse, error) {
	c.log.Info("handle getrate", "base", rr.GetBase(), "destination", rr.GetDestination())

	if rr.Base == rr.Destination {
		// err := status.Errorf(
		// 	codes.InvalidArgument,
		// 	"Base currency %s cannot be the same as the destination currency %s",
		// 	rr.Base.String(),
		// 	rr.Destination.String(),
		// )
		errStatus := status.Newf(
			codes.InvalidArgument,
			"Base currency %s cannot be the same as the destination currency %s",
			rr.Base.String(),
			rr.Destination.String(),
		)

		errStatus, wde := errStatus.WithDetails(rr)
		if wde != nil {
			return nil, wde
		}

		return nil, errStatus.Err()
	}

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &currency.RateResponse{Base: rr.Base, Destination: rr.Destination, Rate: rate}, nil
}

func (c *Currency) SubscribeRates(src currency.Currency_SubscribeRatesServer) error {
	for {
		rr, err := src.Recv()
		if err == io.EOF {
			c.log.Info("Client has closed connection")
			break
		}

		if err != nil {
			c.log.Error("Unable to read from client", "error", err)
			return err
		}

		c.log.Info("Handle client request", "request", rr)

		rrs, ok := c.subscriptions[src]
		if !ok {
			rrs = []*currency.RateRequest{}
		}

		var validationError *status.Status
		for _, v := range rrs {
			if v.Base == rr.Base && v.Destination == rr.Destination {
				validationError = status.Newf(codes.AlreadyExists, "unable to subscribe for currency as subscription already exists")
				validationError, err = validationError.WithDetails(rr)
				if err != nil {
					c.log.Error("Unable to add metadata to error", "error", err)
					break
				}

				break
			}
		}

		if validationError != nil {
			src.Send(
				&currency.StreamingRateResponse{
					Message: &currency.StreamingRateResponse_Error{
						Error: validationError.Proto(),
					},
				},
			)
			continue
		}

		rrs = append(rrs, rr)
		c.subscriptions[src] = rrs
	}

	return nil
}
