// internal/steps/planapply/switcher.go
package planapply

import (
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"
)

// StepPlanApply usa o Switch genérico para rotear baseado em Mode
func StepPlanApply(planChain, applyChain engine.Step) engine.Step {
	return engine.SwitchValue(
		// Função que extrai o valor do state
		func(state *core.ExecutionState) string {
			return state.Mode
		},
		// Map de casos
		map[string]engine.Step{
			"PLAN":  planChain,
			"APPLY": applyChain,
		},
		// Sem default (vai retornar erro se Mode não for PLAN/APPLY)
		nil,
	)
}
