package environment

import (
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/core"

	// Importa a biblioteca que contém os arquivos que você passou
	"github.com/ServerPlace/iac-runner/pkg/environment"
)

// Chave privada para evitar colisão
type envStateKey struct{}

// Helpers de Injeção e Recuperação

// WithEnvironment armazena o objeto detectado no contexto
func WithEnvironment(state *core.ExecutionState, env environment.Environment) {
	state.Set(envStateKey{}, env)
}

// GetEnvironment recupera o objeto do contexto (usado por steps futuros)
func GetEnvironment(state *core.ExecutionState) (environment.Environment, error) {
	val, ok := state.Get(envStateKey{})
	if !ok {
		return environment.Environment{}, fmt.Errorf("nenhum ambiente CI/CD carregado no estado")
	}

	env, ok := val.(environment.Environment)
	if !ok {
		return environment.Environment{}, fmt.Errorf("tipo inválido de ambiente no estado")
	}

	return env, nil
}
