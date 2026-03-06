// internal/steps/planapply/plan.go
package planapply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ServerPlace/iac-controller/pkg/api"
	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/plans"
	"github.com/ServerPlace/iac-runner/internal/domain/terragrunt"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"
	"github.com/ServerPlace/iac-runner/internal/steps/git"
	"github.com/ServerPlace/iac-runner/internal/steps/prepare"
	"github.com/ServerPlace/iac-runner/pkg/cigroup"
	"github.com/ServerPlace/iac-runner/pkg/controller"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

func StepPlan(ctrl controller.Client) engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			cfg := config.Get()

			env, err := environment.GetEnvironment(state)
			if err != nil {
				return err
			}

			queue, err := prepare.GetExecutionQueue(state)
			if err != nil {
				logger.Warn().Msg("Plan: No execution queue found, skipping.")
				return next(ctx, state)
			}

			if len(queue) == 0 {
				logger.Info().Msg("Plan: Queue empty.")
				return next(ctx, state)
			}

			opts := []terragrunt.Option{}
			if cacheDir := cfg.TfPluginCacheDir; cacheDir != "" {
				logger.Info().Str("path", cacheDir).Msg("Setting up plugin cache")
				if err := os.MkdirAll(cacheDir, 0755); err != nil {
					logger.Warn().Err(err).Msg("Failed to create plugin cache dir")
				}
				opts = append(opts, terragrunt.WithPluginCacheDir(cacheDir))
			}

			credentials, err := ctrl.AccessToken(ctx, api.CredentialsRequest{
				Mode:            api.ModePlan,
				Repo:            env.RepoName,
				PRNumber:        env.ChangeNumber,
				HeadSHA:         env.CheckedOutSHA,
				SourceBranch:    env.SourceBranch,
				TargetBranch:    env.TargetBranch,
				SourceBranchSHA: env.SourceBranchSHA,
			})
			if err != nil {
				return err
			}

			tg, err := terragrunt.New(ctx, cfg.TerragruntBin, cfg.TerraformBin,
				append(opts,
					terragrunt.WithCredentials("GOOGLE_OAUTH_ACCESS_TOKEN", credentials.AccessToken),
					terragrunt.WithExtendedPlan(true),
					terragrunt.WithLiveOutput(true),
					terragrunt.WithGrouper(cigroup.New(env.Provider)),
				)...)
			if err != nil {
				return err
			}

			var collectedOutputs []plans.PlanOutput

			for _, item := range queue {
				stackPath := filepath.Join(env.Workspace, item.Path)
				logger.Info().Msgf("🗓️  Planning: %s", item.Path)

				if err := tg.Init(ctx, stackPath); err != nil {
					return err
				}

				var res *terragrunt.Result
				var planErr error

				switch item.Action {
				case "plan":
					res, planErr = tg.Plan(ctx, stackPath)
				case "destroy":
					res, planErr = tg.PlanDestroy(ctx, stackPath)
				}

				if planErr == terragrunt.ErrNochange {
					logger.Info().Msgf("Plan: %s unchanged", item.Path)
					continue
				}

				if planErr != nil {
					return fmt.Errorf("erro no plan de %s: %w", item.Path, planErr)
				}

				if res != nil {
					collectedOutputs = append(collectedOutputs, plans.PlanOutput{
						Path:         item.Path,
						Output:       res.Output,
						PlanFilePath: res.PlanFilePath, // ✅ Propagar path
						JSONPlan:     res.JSONPlan,     // ✅ Propagar bytes
					})
				}
			}

			targets, err := git.GetTargets(state)
			if err != nil {
				return fmt.Errorf("não foi possível recuperar targets do git: %w", err)
			}

			analysis := plans.Analyse(collectedOutputs, targets)
			fmt.Print(analysis.GenerateSummary())

			SetPlan(state, &analysis)

			return next(ctx, state)
		}
	}
}
