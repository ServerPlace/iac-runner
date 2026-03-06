package compliance

import (
	"context"
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"

	// Domain
	"github.com/ServerPlace/iac-runner/internal/domain/compliance"

	// Infra
	infraGcs "github.com/ServerPlace/iac-runner/internal/infrastructure/gcs"
	infraGit "github.com/ServerPlace/iac-runner/internal/infrastructure/git"

	// Steps Inputs
	gitStep "github.com/ServerPlace/iac-runner/internal/steps/git"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"

	"github.com/ServerPlace/iac-runner/pkg/log"
)

// AnalyzeCompliance é o STEP (O Glue Code)
func AnalyzeCompliance() engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			logger.Info().Msg("📋 [Compliance] Analyzing architecture...")

			// 1. Recuperar Dados do Contexto (Inputs)
			env, err := environment.GetEnvironment(state)
			if err != nil {
				return err
			}

			targets, err := gitStep.GetTargets(state)
			if err != nil {
				return err
			}

			// 2. Carregar Registry do GCS
			cfg := config.Get()
			if cfg.ComplianceRegistryURI == "" {
				return fmt.Errorf("compliance: COMPLIANCE_REGISTRY_URI não configurado")
			}

			registryData, err := infraGcs.Download(ctx, cfg.ComplianceRegistryURI)
			if err != nil {
				return fmt.Errorf("compliance: falha ao baixar registry: %w", err)
			}

			registry, err := compliance.LoadRegistry(registryData)
			if err != nil {
				return err
			}

			// 3. Inicializar ContentProvider
			gitProvider := infraGit.NewProvider(env.Workspace)

			// 4. Executar Lógica de Domínio
			analysis, err := compliance.CheckArchitecture(ctx, targets.Files, registry, gitProvider, cfg.ModuleRegistryRoot, targets.BaseSHA)
			if err != nil {
				return err
			}

			// 5. Adaptação de Saída
			if analysis.IsAdminChange {
				logger.Warn().Msg("⚠️  Admin change detected")
			}

			SetStructure(state, analysis)

			logger.Info().Msg("✅ Compliance Checks Passed")
			return next(ctx, state)
		}
	}
}
