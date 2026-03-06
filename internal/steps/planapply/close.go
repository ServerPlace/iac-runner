// internal/steps/planapply/close.go
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

// StepClosePlan notifica o backend que o apply foi concluído e fecha o PR/MR
func StepClosePlan(ctrl controller.Client) engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)

			env, err := environment.GetEnvironment(state)
			if err != nil {
				return fmt.Errorf("ClosePlan: failed to get environment: %w", err)
			}

			if env.ChangeNumber == "" {
				logger.Info().Msg("ClosePlan: not a PR/MR environment, skipping")
				return next(ctx, state)
			}

			req := api.ClosePlanRequest{
				Repo:     env.RepoName,
				PRNumber: mustParsePRNumber(env.ChangeNumber),
				HeadSHA:  env.CheckedOutSHA,
			}

			logger.Info().
				Str("repo", req.Repo).
				Int("pr", req.PRNumber).
				Str("sha", req.HeadSHA).
				Msg("Closing plan on backend")

			resp, err := ctrl.ClosePlan(ctx, req)
			if err != nil {
				return fmt.Errorf("ClosePlan: failed to close plan: %w", err)
			}

			logger.Info().
				Str("deployment_id", resp.DeploymentID).
				Str("status", resp.Status).
				Str("message", resp.Message).
				Msg("Plan closed successfully")

			return next(ctx, state)
		}
	}
}
