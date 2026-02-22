package main

import (
	"github.com/cnpg-broker/pkg/config"
	"github.com/cnpg-broker/pkg/logger"
	"github.com/cnpg-broker/pkg/router"
)

func main() {
	cfg := config.Get()
	logger.Init()
	logger.Info("starting cnpg-broker on port %d", cfg.Port)

	r := router.New()
	logger.Fatal("failed to start HTTP router: %v", r.Start(cfg.Port))
}
