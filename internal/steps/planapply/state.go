package planapply

import (
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/plans"
)

type planStateKey struct{}

func SetPlan(state *core.ExecutionState, result *plans.PlanAnalysis) {
	state.Set(planStateKey{}, result)
}

// GetPlan recupera a análise do plan do estado
func GetPlan(state *core.ExecutionState) (*plans.PlanAnalysis, error) {
	val, ok := state.Get(planStateKey{})
	if !ok {
		return nil, fmt.Errorf("nenhuma análise de plan encontrada no estado")
	}

	analysis, ok := val.(*plans.PlanAnalysis)
	if !ok {
		return nil, fmt.Errorf("tipo inválido no estado para plan analysis")
	}

	return analysis, nil
}
