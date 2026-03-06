package safety

import (
	"context"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"

	// Domain
	domainSafety "github.com/ServerPlace/iac-runner/internal/domain/safety"

	// Infra
	infraGit "github.com/ServerPlace/iac-runner/internal/infrastructure/git"

	gitStep "github.com/ServerPlace/iac-runner/internal/steps/git"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"

	"github.com/ServerPlace/iac-runner/pkg/log"
)

func StepCheckSafety() engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			logger.Info().Msg("🛡️ [Safety] Verifying integrity...")

			// 1. Recuperar Dados do Contexto (Inputs)
			env, err := environment.GetEnvironment(state)
			if err != nil {
				return err
			}

			targets, err := gitStep.GetTargets(state)
			if err != nil {
				return err
			}

			// 2. Inicializar Infraestrutura (Adapter)
			gitProvider := infraGit.NewProvider(env.Workspace)

			// 3. Executar Lógica de Domínio (Business Rule)
			// O step delega 100% da responsabilidade de validação para o domínio
			logger.Debug().Msgf("Targets: %v", string(targets.Json()))
			if err := domainSafety.ValidateChanges(targets, gitProvider, env.Workspace); err != nil {
				logger.Error().Msgf("Safety Check Failed: %v", err)
				return err // Bloqueia a pipeline
			}

			logger.Info().Msg("✅ Safety Checks Passed")
			return next(ctx, state)
		}
	}
}
