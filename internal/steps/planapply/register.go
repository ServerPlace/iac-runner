// internal/steps/planapply/register.go
package planapply

import (
	"context"
	"fmt"

	"github.com/ServerPlace/iac-controller/pkg/api"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"
	"github.com/ServerPlace/iac-runner/pkg/controller"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

// StepRegisterPlan envia o resultado da análise de plan para o backend
func StepRegisterPlan(ctrl controller.Client) engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)

			// 1. Obter análise do plan (gerada pelo StepPlan)
			analysis, err := GetPlan(state)
			if err != nil {
				logger.Warn().Msg("RegisterPlan: No plan analysis found, skipping registration")
				return next(ctx, state)
			}

			// 2. Obter environment
			env, err := environment.GetEnvironment(state)
			if err != nil {
				return fmt.Errorf("failed to get environment: %w", err)
			}

			// 3. Validar que é um ambiente de PR/MR
			if env.ChangeNumber == "" {
				logger.Info().Msg("RegisterPlan: Not a PR/MR environment, skipping registration")
				return next(ctx, state)
			}

			// 4. Extrair lista de stacks do plan
			stacks := make([]string, 0, len(analysis.Components))
			for _, comp := range analysis.Components {
				stacks = append(stacks, comp.Path)
			}

			// 5. Preparar request
			req := api.RegisterPlanRequest{
				Repo:            env.RepoName,
				PRNumber:        mustParsePRNumber(env.ChangeNumber),
				HeadSHA:         env.CheckedOutSHA,
				SourceBranchSHA: env.SourceBranchSHA,
				SourceBranch:    env.SourceBranch,
				TargetBranch:    env.TargetBranch,
				PlanOutput:      analysis.GenerateSummary(),
				Stacks:          stacks,
				User:            env.User, // Direto do environment, sem helper
			}

			// 6. Registrar no backend
			logger.Info().
				Str("repo", req.Repo).
				Int("pr", req.PRNumber).
				Str("sha", req.HeadSHA).
				Int("stacks", len(req.Stacks)).
				Msg("Registering plan with backend")

			resp, err := ctrl.RegisterPlan(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to register plan: %w", err)
			}

			logger.Info().
				Str("deployment_id", resp.DeploymentID).
				Int("plan_version", resp.PlanVersion).
				Str("status", resp.Status).
				Msg("Plan registered successfully")

			return next(ctx, state)
		}
	}
}

// mustParsePRNumber converte string para int
// Panic se não conseguir converter (fail fast - dados corrompidos)
func mustParsePRNumber(changeNumber string) int {
	var prNum int
	if _, err := fmt.Sscanf(changeNumber, "%d", &prNum); err != nil {
		panic(fmt.Sprintf("invalid PR number format: %s", changeNumber))
	}
	return prNum
}
