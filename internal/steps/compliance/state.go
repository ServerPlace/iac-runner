package compliance

import (
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/compliance"
)

type complianceStateKey struct{}

func SetStructure(state *core.ExecutionState, result compliance.StructResult) {
	state.Set(complianceStateKey{}, result)
}

func GetStructure(state *core.ExecutionState) (compliance.StructResult, error) {
	val, ok := state.Get(complianceStateKey{})
	if !ok {
		return compliance.StructResult{}, fmt.Errorf("sem analise no estado")
	}
	s, ok := val.(compliance.StructResult)
	if !ok {
		return compliance.StructResult{}, fmt.Errorf("tipo inválido de analise no estado")
	}
	return s, nil
}
