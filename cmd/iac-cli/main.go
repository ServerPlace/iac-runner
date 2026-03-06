package main

import (
	"context"
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/workflows"
	"github.com/ServerPlace/iac-runner/pkg/controller"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"github.com/ServerPlace/iac-runner/pkg/version"
	"os"
)

func main() {

	// ====================================================================
	// 1. BootStrap (Start Engine)
	// ====================================================================

	// Detect pipeline mode
	mode := os.Getenv("IAC_MODE")
	if mode == "" {
		mode = "PLAN" // Default seguro
	}
	state := core.NewState(mode)

	// Init Config Singleton
	cfg := config.Get()

	// Global Context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Start Logger
	logger := log.New(log.Setup())
	ctx = log.WithLogger(ctx, logger)
	logger.Info().Msg("Executor starting")

	// startInfo
	logger.Info().Msgf(
		"\n***************************************************\n"+
			" Linuxplace IAC CLI \n Version: %s \n Build time: %s\n"+
			"***************************************************", version.Version, version.BuildTime)
	logger.Info().Msgf(`Config:
Timeout: %s
`, cfg.Timeout)
	// InitController
	controllerSecret := os.Getenv("CONTROLLER_SECRET")
	iacController, _ := controller.NewController(ctx, cfg.ControllerUrl, controllerSecret)
	// ====================================================================
	// 2. Build pipeline
	// ====================================================================
	pipeline := workflows.BuildGlobalWorkflow(logger, iacController)
	// ====================================================================
	// 3. Run Cli
	// ====================================================================

	// Transformamos a pipeline em um executor final.
	// O 'engine.NoOp' é a função que será chamada quando o último passo terminar.
	runner := pipeline(engine.NoOp)

	// Execute
	if err := runner(ctx, state); err != nil {
		// Erro Fatal: A pipeline quebrou no meio.
		// Usamos Fprintf no Stderr para garantir que pipelines de CI peguem o erro.
		fmt.Fprintf(os.Stderr, "\n❌ CRITICAL FAILURE: %v\n", err)
		os.Exit(1)
	}
}
