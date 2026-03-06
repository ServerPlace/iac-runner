package engine

import (
	"context"
	"github.com/ServerPlace/iac-runner/internal/core"
)

type Executable func(ctx context.Context, state *core.ExecutionState) error
type Step func(next Executable) Executable

// Build Execution Stack
func Pipeline(steps ...Step) Step {
	return func(final Executable) Executable {
		for i := len(steps) - 1; i >= 0; i-- {
			final = steps[i](final)
		}
		return final
	}
}

func NoOp(ctx context.Context, state *core.ExecutionState) error { return nil }
