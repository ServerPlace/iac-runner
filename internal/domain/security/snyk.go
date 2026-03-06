// internal/domain/security/snyk.go
package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/ServerPlace/iac-runner/pkg/command"
)

var (
	ErrNoApiToken = errors.New("snyk API token not provided")
)

type SnykScanner struct {
	binPath string
	token   string
}

func NewSnykScanner(binPath, token string) *SnykScanner {
	if binPath == "" {
		binPath = "snyk"
	}
	return &SnykScanner{
		binPath: binPath,
		token:   token,
	}
}

func (s *SnykScanner) Name() string {
	return "snyk"
}

// ScanPlanFiles recebe paths de arquivos no disco
func (s *SnykScanner) ScanPlanFiles(ctx context.Context, planFiles []string) (*ScanResult, error) {
	result := &ScanResult{
		Scanner: s.Name(),
		ByPath:  make(map[string]PathScan),
	}

	for _, planFile := range planFiles {
		// ✅ Comando correto do Snyk
		params := []string{
			"iac", "test",
			planFile,                  // ✅ Arquivo primeiro
			"--scan=resource-changes", // ✅ Escaneia apenas mudanças (padrão recomendado)
			"--json",                  // ✅ Output em JSON
		}

		opts := command.RunOptions{
			LiveOutput: true, // ✅ Mostrar progresso na tela
		}

		// ✅ Token via env var
		if s.token != "" {
			return nil, ErrNoApiToken
		}

		output, err := command.Run(ctx, s.binPath, params, opts)

		// ✅ Tratamento de exit code
		// Snyk retorna:
		// - 0: sem issues
		// - 1: issues encontradas (não é erro!)
		// - 2: failure (erro de autenticação, etc)
		// - 3: no supported projects
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exitCode := ee.ExitCode()
				if exitCode == 1 {
					// Exit code 1 = issues encontradas, continuar
					err = nil
				} else if exitCode == 3 {
					// Exit code 3 = nenhum projeto suportado (pular)
					continue
				} else {
					// Exit code 2+ = erro real
					return nil, fmt.Errorf("snyk scan failed for %s (exit %d): %w",
						planFile, exitCode, err)
				}
			} else {
				return nil, fmt.Errorf("snyk scan failed for %s: %w", planFile, err)
			}
		}

		result.RawOutput += output

		pathResult, err := s.parseOutput([]byte(output))
		if err != nil {
			return nil, fmt.Errorf("failed to parse snyk output for %s: %w", planFile, err)
		}

		pathResult.Path = planFile
		result.ByPath[planFile] = pathResult

		// Acumular totais
		result.TotalIssues += len(pathResult.Issues)
		for _, issue := range pathResult.Issues {
			switch issue.Severity {
			case SeverityCritical:
				result.Critical++
			case SeverityHigh:
				result.High++
			case SeverityMedium:
				result.Medium++
			case SeverityLow:
				result.Low++
			}
		}
	}

	return result, nil
}

// snykOutput representa a estrutura JSON do Snyk
// Baseado na documentação oficial: https://docs.snyk.io/snyk-cli/commands/iac-test
type snykOutput struct {
	// Snyk IaC test retorna issues aqui
	InfrastructureAsCodeIssues []snykIssue `json:"infrastructureAsCodeIssues"`

	// Metadata do scan
	OK          bool `json:"ok"`
	TestSummary struct {
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
		Critical int `json:"critical"`
	} `json:"testSummary"`
}

type snykIssue struct {
	ID          string `json:"id"`          // Ex: SNYK-CC-TF-1, SNYK-CC-AWS-422
	Title       string `json:"title"`       // Título descritivo
	Severity    string `json:"severity"`    // critical, high, medium, low
	Msg         string `json:"msg"`         // Mensagem detalhada
	Issue       string `json:"issue"`       // Descrição do problema
	Impact      string `json:"impact"`      // Impacto potencial
	Resolve     string `json:"resolve"`     // Como resolver
	Remediation string `json:"remediation"` // Passos de remediação

	// Informações do recurso
	Resource string   `json:"resource"` // Nome do recurso
	Subtype  string   `json:"subtype"`  // Tipo (aws_s3_bucket, etc)
	Path     []string `json:"path"`     // Path no Terraform

	// Detalhes técnicos
	References []string `json:"references"` // Links de referência
	IsIgnored  bool     `json:"isIgnored"`  // Se foi ignorado
	IgnoredBy  string   `json:"ignoredBy"`  // Quem ignorou
}

func (s *SnykScanner) parseOutput(data []byte) (PathScan, error) {
	var output snykOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return PathScan{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	scan := PathScan{
		Issues: []Issue{},
	}

	for _, snykIssue := range output.InfrastructureAsCodeIssues {
		// ✅ Pular issues ignorados
		if snykIssue.IsIgnored {
			continue
		}

		severity := parseSeverity(snykIssue.Severity)

		issue := Issue{
			ID:           snykIssue.ID,
			Title:        snykIssue.Title,
			Severity:     severity,
			ResourceType: snykIssue.Subtype,
			ResourceName: snykIssue.Resource,
			Description:  snykIssue.Msg,
		}

		// ✅ Combinar informações úteis na descrição
		if snykIssue.Impact != "" {
			issue.Description += fmt.Sprintf("\nImpact: %s", snykIssue.Impact)
		}
		if snykIssue.Resolve != "" {
			issue.Description += fmt.Sprintf("\nResolve: %s", snykIssue.Resolve)
		}

		scan.Issues = append(scan.Issues, issue)

		if severity == SeverityCritical {
			scan.HasCritical = true
		}

		if scan.Severity == "" || severity > scan.Severity {
			scan.Severity = severity
		}
	}

	return scan, nil
}
