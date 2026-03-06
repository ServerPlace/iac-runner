package git

import (
	"context"
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"

	// Importa o domínio para usar a lógica pura (Aliasing para clareza)
	domainGit "github.com/ServerPlace/iac-runner/internal/domain/git"

	// Importa o step de environment para ler o contexto dele
	"github.com/ServerPlace/iac-runner/internal/steps/environment"

	"github.com/ServerPlace/iac-runner/pkg/log"
	"github.com/rs/zerolog"
)

const (
	DebugChange string = "Change: %v, From: %s, To: %s"
)

func StepDetectChanges(detector domainGit.ChangeDetector) engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			logger.Info().Msg("🔍 [Git] Detecting Changes...")

			// 1. Obtém dados do Step anterior (Environment)
			env, err := environment.GetEnvironment(state)
			if err != nil {
				logger.Fatal().Msg("Fail to recover environment")
				return fmt.Errorf("fail to recover enviroment %w", err)
			}

			if env.Workspace == "" || env.CheckedOutSHA == "" {
				return fmt.Errorf("invalid environment data: workspace or sha empty")
			}
			cfg := config.Get()
			// 2. Usa a Lógica de Domínio (Pure Logic) para resolver SHA
			if env.TargetBranch == "" {
				env.TargetBranch = cfg.BaseBranch
			}
			if logger.GetLevel() == zerolog.DebugLevel {
				logger.Debug().Msgf("Workspace: %s", env.Workspace)
				v, _ := domainGit.ListLocalBranches(env.Workspace)
				branches := ""
				for _, v := range v {
					branches += fmt.Sprintf("- %s\n", v)
				}
				logger.Debug().Msgf("Local Branches:\n %s", branches)
			}
			baseSHA, err := domainGit.ResolveBranchTip(env.Workspace, env.TargetBranch)
			if err != nil {
				logger.Error().Err(err).Msg("failed to resolve base SHA")
				return err
			}
			sourceSHA, err := domainGit.ResolveBranchTip(env.Workspace, env.SourceBranch)
			if err != nil {
				logger.Error().Err(err).Msg("failed to resolve feature branch SHA")
				return err
			}
			// 3. Executa a detecção (Injeção de dependência)
			logger.Debug().Msgf("Workspace: %s, Base SHA: %s, CheckedOutSHA: %s", env.Workspace, baseSHA, env.CheckedOutSHA)
			target, err := detector.Detect(ctx, env.Workspace, baseSHA, env.CheckedOutSHA)
			if err != nil {
				logger.Err(err).Msg("fail to detect detectors")
				return fmt.Errorf("fail to detect detectors %w", err)
			}
			target.SourceSHA = sourceSHA
			// 4. Log (Orquestração)
			if logger.GetLevel() == zerolog.DebugLevel && len(target.Files) > 0 {
				var debugStr string
				for _, v := range target.Files {
					debugStr += fmt.Sprintf(DebugChange, v.ChangeType, v.OldPath, v.Path)
				}
				logger.Debug().Msg(debugStr)
			}

			// 5. Salva no Contexto (Mecanismo do Step)
			// Usa a função definida no context.go DENTRO deste pacote steps/git
			SetTargets(state, &target)

			return next(ctx, state)
		}
	}
}
