package prepare

import (
	"context"
	"strings"

	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/ServerPlace/iac-runner/internal/core"

	// Importa o domínio renomeado
	domainPrepare "github.com/ServerPlace/iac-runner/internal/domain/prepare"

	gitStep "github.com/ServerPlace/iac-runner/internal/steps/git"
	"github.com/ServerPlace/iac-runner/internal/domain/compliance"
	"github.com/ServerPlace/iac-runner/internal/domain/git"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

func StepPrepare() engine.Step {
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)
			logger.Info().Msg("⚙️ [Prepare] Resolving Dependencies & Resurrecting Files...")

			env, err := environment.GetEnvironment(state)
			if err != nil {
				return err
			}
			targets, err := gitStep.GetTargets(state)
			if err != nil {
				return err
			}

			cfg := config.Get()
			baseRef := cfg.BaseBranch
			if baseRef == "" {
				baseRef = "HEAD^"
			}
			// Try To Filter Invalid Target files off
			filteredFiles := make([]git.FileChange, 0)
			for _, t := range targets.Files {
				isAllowed := true
				for _, structural := range compliance.StructuralFiles {
					if strings.HasSuffix(t.Path, structural) {
						isAllowed = false
						break
					}
				}
				if isAllowed {
					filteredFiles = append(filteredFiles, t)
				}
			}
			targets.Files = filteredFiles

			// Lógica Pura
			svc := domainPrepare.New(env.Workspace, baseRef)
			destroyDirs, applyDirs, err := svc.BuildExecutionQueue(targets)
			if err != nil {
				return err
			}

			// Monta Fila Unificada
			var queue []ExecutionItem

			// 1. Destroys
			for _, d := range destroyDirs {
				queue = append(queue, ExecutionItem{Path: d, Action: "destroy"})
			}

			// 2. Applies/Plans
			action := "apply"
			if state.Mode == "PLAN" {
				action = "plan"
			}

			for _, a := range applyDirs {
				queue = append(queue, ExecutionItem{Path: a, Action: action})
			}

			if len(queue) > 0 {
				logger.Info().Msgf("📋 Prepared Queue: %d items", len(queue))
				for _, item := range queue {
					logger.Debug().Msgf("   -> %s %s", item.Action, item.Path)
				}
			} else {
				logger.Warn().Msg("Empty execution queue (nothing to do)")
			}
			SetExecutionQueue(state, queue)
			return next(ctx, state)
		}
	}
}
