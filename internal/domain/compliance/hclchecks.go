package compliance

import (
	"fmt"
	iHcl "github.com/ServerPlace/iac-runner/internal/infrastructure/hcl"
	"github.com/hashicorp/hcl/v2"
	"sort"
	"strings"
)

// ===============================
// Validator + Options Pattern
// ===============================

type CheckFunc func(a *iHcl.Analisys) hcl.Diagnostics

type Validator struct {
	analysis *iHcl.Analisys
	checks   []CheckFunc
}

type Option func(v *Validator)

// NewValidator inicializa parser, parseia o arquivo e registra checks via options.
// Você chama: NewValidator("terragrunt.hcl", WithX(...), WithY(...))
func NewValidator(filename string, opts ...Option) *Validator {
	a := iHcl.NewAnalisysFromFile(filename)

	v := &Validator{
		analysis: a,
		checks:   make([]CheckFunc, 0, 8),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(v)
		}
	}

	return v
}

// NewValidator inicializa parser, parseia o arquivo e registra checks via options.
// Você chama: NewValidator("terragrunt.hcl", WithX(...), WithY(...))
func NewValidatorFromBytes(content []byte, filename string, opts ...Option) *Validator {
	a := iHcl.NewAnalisysFromBytes(content, filename)

	v := &Validator{
		analysis: a,
		checks:   make([]CheckFunc, 0, 8),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(v)
		}
	}

	return v
}

// Run executa: (1) erros de parse, (2) checks em ordem. Fail-fast no primeiro erro.
func (v *Validator) Run() hcl.Diagnostics {
	if v.analysis == nil {
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Validator inválido",
			Detail:   "analysis é nil",
		}}
	}

	// 1) Parse errors
	if v.analysis.Diagnostics.HasErrors() {
		return v.analysis.Diagnostics
	}

	// 2) Checks (fail-fast)
	for _, chk := range v.checks {
		diags := chk(v.analysis)
		if diags.HasErrors() {
			return diags
		}
	}

	return nil
}

// WithCustomCheck permite plugar regras arbitrárias sem criar nova option pronta.
func WithCustomCheck(fn CheckFunc) Option {
	return func(v *Validator) {
		v.checks = append(v.checks, fn)
	}
}

// ===============================
// Options prontas (exemplos)
// ===============================

// WithProhibitedInputKeys proíbe a presença de certas CHAVES em inputs.
// Ex: WithProhibitedInputKeys("project_id", "location")
func WithProhibitedInputKeys(keys ...string) Option {
	// Normaliza para comparação
	blocked := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		blocked[k] = struct{}{}
	}

	return WithCustomCheck(func(a *iHcl.Analisys) hcl.Diagnostics {
		var diags hcl.Diagnostics

		inputs, d := a.ParseInputs()
		if d.HasErrors() {
			return d // fail-fast
		}

		// Determinismo na iteração
		inputKeys := make([]string, 0, len(inputs))
		for k := range inputs {
			inputKeys = append(inputKeys, k)
		}
		sort.Strings(inputKeys)

		for _, k := range inputKeys {
			if _, forbidden := blocked[k]; !forbidden {
				continue
			}

			expr := inputs[k]
			rng := expr.Range()

			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Input proibido por policy",
				Detail:   fmt.Sprintf("A chave inputs.%s é proibida por policy.", k),
				Subject:  &rng,
			})
			return diags // FAIL-FAST no primeiro erro
		}

		return diags
	})
}
