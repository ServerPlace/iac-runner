package terragrunt

import (
	"context"
)

type Execution struct {
	basePath        string
	Component       string
	ClouEnvironment string
	Project         string
	Region          string
	Account         string
}
type Client interface {
	Init(ctx context.Context, basePath string) (Execution, error)
}
