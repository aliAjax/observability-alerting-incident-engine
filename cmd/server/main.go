package main

import (
	"context"
	"fmt"
	"os"

	"github.com/observability-alerting/engine/internal/bootstrap"
	"github.com/observability-alerting/engine/internal/config"
)

func main() {
	cfgPath := ""
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	logger := bootstrap.NewLogger(cfg.LogLevel)
	app, err := bootstrap.New(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create app: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "run app: %v\n", err)
		os.Exit(1)
	}
}
