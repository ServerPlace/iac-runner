package environment

import (
	"context"
	"fmt"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"github.com/rs/zerolog"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/pkg/environment"
)

const (
	debugTrace string = `
   ✅ Provider: %s
   📝 Event:    %s
%s
`
)

// StepLoadEnvironment executa a detecção automática do CI provider
func StepLoadEnvironment() engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			l := log.FromContext(ctx)
			l.Info().Msg("🌍 [Env] Discover Execution Environment...")

			// 1. Chama a lógica pura que você forneceu
			envData, err := environment.Setup()
			if l.GetLevel() == zerolog.DebugLevel {
				l.Debug().Msgf("Environment data: %+v", envData)
			}
			l.Info().Msgf("Running in %s%s%s environment", log.Bold, envData.Provider, log.Reset)
			if err != nil {
				// Falha crítica: Se não sabemos onde estamos, não podemos prosseguir
				l.Err(err).Msg("could not detect environment")
				return fmt.Errorf("could not detect environment: %w", err)
			}

			// 2. Logs de rastreabilidade (Útil para debug no CI)

			optString := ""
			if envData.RepoName != "" {
				optString += fmt.Sprintf("   📦 Repo:     %s\n", envData.RepoName)
			}
			if envData.ChangeNumber != "" {
				optString += fmt.Sprintf("   🔀 PR/MR:    %s\n", envData.ChangeNumber)
			}
			l.Debug().Msgf(debugTrace,
				envData.Provider,
				envData.Event,
				optString)
			// 3. Inject Env
			WithEnvironment(state, envData)

			// 4. Next
			return next(ctx, state)
		}
	}
}
