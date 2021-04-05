package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/grkmk/glm-currency/data"
	protos "github.com/grkmk/glm-currency/protos/currency"

	"github.com/grkmk/glm-currency/server"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := hclog.New(&hclog.LoggerOptions{
		Name:  "[product-currency]",
		Level: hclog.DefaultLevel,
		Color: hclog.AutoColor,
	})

	rates, err := data.NewRates(log)
	if err != nil {
		log.Error("unable to generate rates", "error", err)
		os.Exit(1)
	}

	gs := grpc.NewServer()
	cs := server.NewCurrency(rates, log)

	protos.RegisterCurrencyServer(gs, cs)

	reflection.Register(gs)

	go func() {
		log.Info("Starting server on port 9092")

		l, err := net.Listen("tcp", ":9092")
		if err != nil {
			log.Error("Unable to listen", "error", err)
			os.Exit(1)
		}

		gs.Serve(l)
	}()

	// trap sigterm or interrupt & gracefully shutdown server
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	signal.Notify(signalChannel, syscall.SIGTERM)

	sig := <-signalChannel
	log.Info("Received terminate, gracefully shutting down", sig)
	gs.GracefulStop()
}
