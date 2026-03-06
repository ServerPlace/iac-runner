package prepare

import (
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/core"
	"path/filepath"
)

// ExecutionItem define uma unidade de trabalho para o Terragrunt
type ExecutionItem struct {
	Path   string // Caminho relativo do diretório da stack
	Action string // "plan", "apply" ou "destroy"
}

// Chave privada e segura para o mapa de estado
type queueStateKey struct{}

// SetExecutionQueue salva a fila ordenada no plano de dados (State)
func SetExecutionQueue(state *core.ExecutionState, queue []ExecutionItem) {
	state.Set(queueStateKey{}, queue)
}

// GetExecutionQueue recupera a fila para uso no Plan/Apply
func GetExecutionQueue(state *core.ExecutionState) ([]ExecutionItem, error) {
	val, ok := state.Get(queueStateKey{})
	if !ok {
		return nil, fmt.Errorf("nenhuma fila de execução encontrada no estado")
	}

	queue, ok := val.([]ExecutionItem)
	if !ok {
		return nil, fmt.Errorf("tipo inválido na fila de execução do estado")
	}

	return queue, nil
}

type Items []ExecutionItem

func (i Items) Len() int { return len(i) }
func (i Items) Stacks(workspace string) []string {
	stacks := make([]string, 0, len(i))
	for _, item := range i {
		stacks = append(stacks, filepath.Join(workspace, item.Path))
	}
	return stacks
}
