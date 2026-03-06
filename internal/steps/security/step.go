// internal/steps/security/step.go
package security

import (
	"context"
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/security"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/planapply"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

// StepScanPlan executa security scan nos plan files
// Se auto-configura baseado em feature flags
func StepScanPlan() engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			cfg := config.Get()

			// Factory do domain cria o scanner
			scanner := security.BuildScanner(logger, cfg)
			if scanner == nil {
				logger.Debug().Msg("⏭️  Security scan disabled")
				return next(ctx, state)
			}

			policy := security.GetPolicy(cfg)

			logger.Info().Msgf("🔒 [Security] Running %s scan...", scanner.Name())

			// Recuperar análise do plan
			analysis, err := planapply.GetPlan(state)
			if err != nil {
				logger.Warn().Msg("No plan analysis found, skipping")
				return next(ctx, state)
			}

			// ✅ Coletar paths dos plan files
			planFiles := make([]string, 0, len(analysis.Components))
			for _, comp := range analysis.Components {
				if comp.PlanFilePath != "" {
					planFiles = append(planFiles, comp.PlanFilePath)
				}
			}

			if len(planFiles) == 0 {
				logger.Info().Msg("No plan files to scan")
				return next(ctx, state)
			}

			logger.Info().Msgf("Scanning %d plan files", len(planFiles))

			// ✅ Scanners leem direto do disco
			result, err := scanner.ScanPlanFiles(ctx, planFiles)
			if err != nil {
				return fmt.Errorf("security scan failed: %w", err)
			}

			// Aplicar política
			result.ApplyPolicy(policy)

			// Log resultado
			fmt.Print(result.GenerateSummary())

			if result.TotalIssues > 0 {
				logger.Warn().Msgf("Found %d security issues", result.TotalIssues)
				for path, pathScan := range result.ByPath {
					if len(pathScan.Issues) > 0 {
						logger.Warn().Msgf("  %s: %d issues", path, len(pathScan.Issues))
						for _, issue := range pathScan.Issues {
							logger.Debug().
								Str("id", issue.ID).
								Str("severity", string(issue.Severity)).
								Str("resource", issue.ResourceName).
								Msgf("    - %s", issue.Title)
						}
					}
				}
			}

			// Salvar no state
			SetSecurityResult(state, result)

			// Falhar se não passou
			if !result.Passed {
				return fmt.Errorf("security scan failed: %s", result.FailureReason)
			}

			logger.Info().Msg("✅ Security scan passed")
			return next(ctx, state)
		}
	}
}
