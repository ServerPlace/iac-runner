// internal/steps/security/state.go
package security

import (
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/security"
)

type securityStateKey struct{}

// SetSecurityResult salva o resultado do scan no state
func SetSecurityResult(state *core.ExecutionState, result *security.ScanResult) {
	state.Set(securityStateKey{}, result)
}

// GetSecurityResult recupera o resultado do scan do state
func GetSecurityResult(state *core.ExecutionState) (*security.ScanResult, error) {
	val, ok := state.Get(securityStateKey{})
	if !ok {
		return nil, fmt.Errorf("nenhum resultado de security scan encontrado no estado")
	}

	result, ok := val.(*security.ScanResult)
	if !ok {
		return nil, fmt.Errorf("tipo inválido no estado para security result")
	}

	return result, nil
}
