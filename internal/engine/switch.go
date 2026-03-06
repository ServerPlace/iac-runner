// internal/engine/switch.go
package engine

import (
	"context"
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

// Case representa um caso em um switch
type Case struct {
	Name      string                          // Nome do caso (para logs)
	Condition func(*core.ExecutionState) bool // Condição
	Chain     Step                            // Step a executar se condição = true
}

// Switch executa diferentes chains baseado em condições
// Executa o PRIMEIRO caso cuja condição seja true
// Se nenhuma condição for true, executa defaultChain (se fornecido)
func Switch(cases []Case, defaultChain Step) Step {
	return func(next Executable) Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)

			// Avaliar casos em ordem
			for _, c := range cases {
				if c.Condition(state) {
					logger.Info().Msgf("🔀 [Switch] Branch: %s", c.Name)
					return c.Chain(next)(ctx, state)
				}
			}

			// Nenhuma condição bateu
			if defaultChain != nil {
				logger.Info().Msg("🔀 [Switch] Branch: default")
				return defaultChain(next)(ctx, state)
			}

			// Sem default, retorna erro
			logger.Error().Msg("🔀 [Switch] No matching case and no default provided")
			return fmt.Errorf("no matching case in switch")
		}
	}
}

// SwitchValue é um switch baseado em valor (como state.Mode)
// Mais conveniente quando você tem um enum/string para comparar
func SwitchValue[T comparable](
	getValue func(*core.ExecutionState) T,
	cases map[T]Step,
	defaultChain Step,
) Step {
	return func(next Executable) Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			logger := log.FromContext(ctx)

			value := getValue(state)

			if chain, ok := cases[value]; ok {
				logger.Info().Msgf("🔀 [Switch] Branch: %v", value)
				return chain(next)(ctx, state)
			}

			if defaultChain != nil {
				logger.Info().Msgf("🔀 [Switch] Branch: default (value=%v)", value)
				return defaultChain(next)(ctx, state)
			}

			return fmt.Errorf("no matching case for value: %v", value)
		}
	}
}
