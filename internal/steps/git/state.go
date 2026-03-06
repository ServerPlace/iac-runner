package git

import (
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/domain/git"
)

// 1. Chave privada e segura para o mapa
type targetsStateKey struct{}

// 2. SetTargets: Escreve no estado mutável
func SetTargets(state *core.ExecutionState, target *git.Target) {
	state.Set(targetsStateKey{}, target)
}

// 3. GetTargets: Lê do estado com Type Assertion segura
func GetTargets(state *core.ExecutionState) (*git.Target, error) {
	val, ok := state.Get(targetsStateKey{})
	if !ok {
		return nil, fmt.Errorf("nenhum target encontrado no estado da execução")
	}

	t, ok := val.(*git.Target)
	if !ok {
		return nil, fmt.Errorf("tipo inválido no estado para targets")
	}
	return t, nil
}
