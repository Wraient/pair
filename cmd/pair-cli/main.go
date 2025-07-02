package main

import (
	// "fmt"
	"github.com/wraient/pair/pkg/appcore"
	"github.com/wraient/pair/pkg/config"
	"github.com/wraient/pair/pkg/logger"
	// "go.uber.org/zap"
)

func main() {
	logger.Initialize(true)
	logger.Info("Pair CLI started")

	config.Initialize()

	// logger.Info("UI mode", zap.String("mode", string(config.Get().UI.Mode)))

	appcore.Start()

}
