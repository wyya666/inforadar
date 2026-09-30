package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"inforadar/internal/application"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := application.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
