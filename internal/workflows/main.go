// internal/workflows/main.go
package workflows

import (
	stepLogger "github.com/ServerPlace/iac-runner/internal/steps/logger"
	securityStep "github.com/ServerPlace/iac-runner/internal/steps/security"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/detectors/hcl"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/internal/steps/compliance"
	"github.com/ServerPlace/iac-runner/internal/steps/environment"
	"github.com/ServerPlace/iac-runner/internal/steps/git"
	"github.com/ServerPlace/iac-runner/internal/steps/planapply"
	"github.com/ServerPlace/iac-runner/internal/steps/prepare"
	"github.com/ServerPlace/iac-runner/internal/steps/safety"
	"github.com/ServerPlace/iac-runner/pkg/controller"
	"github.com/rs/zerolog"
)

// BuildGlobalWorkflow constrói o workflow principal do sistema
// Features são auto-configuradas via factories baseado em feature flags
func BuildGlobalWorkflow(logger zerolog.Logger, client controller.Client) engine.Step {
	return engine.Pipeline(
		// Inicialização
		stepLogger.StepInit(stepLogger.WithLoggerInstance(logger)),
		environment.StepLoadEnvironment(),

		// Análise de mudanças
		git.StepDetectChanges(hcl.NewHCLDetector()),

		// Validações
		compliance.AnalyzeCompliance(),
		safety.StepCheckSafety(),

		// Preparação
		prepare.StepPrepare(),

		// Roteamento Plan/Apply
		engine.SwitchValue(
			func(state *core.ExecutionState) string { return state.Mode },
			map[string]engine.Step{
				"PLAN":  BuildPlanChain(logger, client),
				"APPLY": BuildApplyChain(logger, client),
			},
			nil,
		),
	)
}

// BuildPlanChain constrói a cadeia de steps para o modo PLAN
func BuildPlanChain(logger zerolog.Logger, client controller.Client) engine.Step {
	return engine.Pipeline(
		// Executar terraform plan
		planapply.StepPlan(client),

		// ✅ Security scan (auto-configura via feature flags)
		securityStep.StepScanPlan(),

		// ✅ Registrar plan no backend (auto-configura via feature flags)
		planapply.StepRegisterPlan(client),

		// TODO: Adicionar outros steps opcionais aqui
		// - StepCostEstimation()
		// - StepPRComment()
	)
}

// BuildApplyChain constrói a cadeia de steps para o modo APPLY
func BuildApplyChain(logger zerolog.Logger, client controller.Client) engine.Step {
	return engine.Pipeline(
		// Executar terraform apply
		planapply.StepApply(client),

		// Fechar PR/MR no backend após apply bem-sucedido
		planapply.StepClosePlan(client),
	)
}
