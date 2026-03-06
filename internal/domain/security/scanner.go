// internal/domain/security/scanner.go
package security

import (
	"context"
	"fmt"
)

// Scanner é a interface que implementações concretas devem seguir
type Scanner interface {
	Name() string

	// ✅ ScanPlanFiles recebe paths de arquivos, não bytes
	ScanPlanFiles(ctx context.Context, planFiles []string) (*ScanResult, error)
}

// ScanResult representa o resultado consolidado da análise
type ScanResult struct {
	Scanner       string
	TotalIssues   int
	Critical      int
	High          int
	Medium        int
	Low           int
	ByPath        map[string]PathScan
	RawOutput     string
	Passed        bool
	FailureReason string
}

// PathScan representa o resultado de análise de um path específico
type PathScan struct {
	Path        string
	Issues      []Issue
	Severity    Severity
	HasCritical bool
}

// Issue representa uma vulnerabilidade encontrada
type Issue struct {
	ID           string
	Title        string
	Severity     Severity
	ResourceType string
	ResourceName string
	Description  string
}

// Severity representa o nível de severidade
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"
)

// Policy define os critérios de aprovação
type Policy struct {
	BlockOnCritical bool
	BlockOnHigh     bool
	MaxIssues       int
}

// DefaultPolicy retorna a política padrão (mais restritiva)
func DefaultPolicy() Policy {
	return Policy{
		BlockOnCritical: true,
		BlockOnHigh:     false,
		MaxIssues:       0,
	}
}

// ApplyPolicy aplica a política ao resultado e determina se passou
func (r *ScanResult) ApplyPolicy(p Policy) {
	r.Passed = true

	if p.BlockOnCritical && r.Critical > 0 {
		r.Passed = false
		r.FailureReason = fmt.Sprintf("found %d critical vulnerabilities", r.Critical)
		return
	}

	if p.BlockOnHigh && r.High > 0 {
		r.Passed = false
		r.FailureReason = fmt.Sprintf("found %d high severity vulnerabilities", r.High)
		return
	}

	if p.MaxIssues > 0 && r.TotalIssues > p.MaxIssues {
		r.Passed = false
		r.FailureReason = fmt.Sprintf("found %d issues, maximum allowed is %d", r.TotalIssues, p.MaxIssues)
		return
	}
}

// GenerateSummary gera um resumo legível do scan
func (r *ScanResult) GenerateSummary() string {
	status := "✅ PASSED"
	if !r.Passed {
		status = "❌ FAILED"
	}

	summary := fmt.Sprintf(`
🔒 Security Scan Results (%s)
%s

Total Issues: %d
├─ Critical: %d
├─ High:     %d
├─ Medium:   %d
└─ Low:      %d
`,
		r.Scanner,
		status,
		r.TotalIssues,
		r.Critical,
		r.High,
		r.Medium,
		r.Low,
	)

	if !r.Passed {
		summary += fmt.Sprintf("\n⚠️  Reason: %s\n", r.FailureReason)
	}

	return summary
}
