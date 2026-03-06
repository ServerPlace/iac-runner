package planapply

import (
	"context"
	"github.com/ServerPlace/iac-controller/pkg/api"
	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/terragrunt"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"
	"github.com/ServerPlace/iac-runner/internal/steps/prepare"
	"github.com/ServerPlace/iac-runner/pkg/cigroup"
	"github.com/ServerPlace/iac-runner/pkg/controller"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"os"
	"path/filepath"
)

func StepApply(ctrl controller.Client) engine.Step {
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
				Mode:            api.ModeApply,
				Repo:            env.RepoName,
				PRNumber:        env.ChangeNumber,
				HeadSHA:         env.CheckedOutSHA,
				SourceBranchSHA: env.SourceBranchSHA,
				SourceBranch:    env.SourceBranch,
				TargetBranch:    env.TargetBranch,
				Stacks:          prepare.Items(queue).Stacks(env.Workspace),
				JobID:           "TODO",
				JobToken:        "",
			})
			if err != nil {
				return err
			}

			tg, err := terragrunt.New(ctx, cfg.TerragruntBin, cfg.TerraformBin,
				terragrunt.WithCredentials("GOOGLE_OAUTH_ACCESS_TOKEN", credentials.AccessToken),
				terragrunt.WithLiveOutput(true),
				terragrunt.WithGrouper(cigroup.New(env.Provider)))
			if err != nil {
				return err
			}

			// Execução do Loop
			for _, item := range queue {
				stackPath := filepath.Join(env.Workspace, item.Path)
				logger.Info().Msgf("🏗️  Applying: %s", item.Path)

				if err := tg.Init(ctx, stackPath); err != nil {
					return err
				}
				switch item.Action {
				case "apply":
					if _, err := tg.Apply(ctx, stackPath); err != nil {
						return err
					}
				case "destroy":
					if _, err := tg.Destroy(ctx, stackPath); err != nil {
						return err
					}
				}
			}
			// SÓ CHAMA O NEXT DEPOIS DE TERMINAR TODA A FILA
			return next(ctx, state)
		}
	}
}
