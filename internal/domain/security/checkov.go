// internal/domain/security/checkov.go
package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type CheckovScanner struct {
	binPath string
}

func NewCheckovScanner(binPath string) *CheckovScanner {
	if binPath == "" {
		binPath = "checkov"
	}
	return &CheckovScanner{binPath: binPath}
}

func (c *CheckovScanner) Name() string {
	return "checkov"
}

// ScanPlanFiles recebe paths de arquivos no disco
func (c *CheckovScanner) ScanPlanFiles(ctx context.Context, planFiles []string) (*ScanResult, error) {
	result := &ScanResult{
		Scanner: c.Name(),
		ByPath:  make(map[string]PathScan),
	}

	for _, planFile := range planFiles {
		// Checkov lê direto do arquivo no disco
		cmd := exec.CommandContext(ctx, c.binPath,
			"--framework", "terraform_plan",
			"--file", planFile,
			"--output", "json",
			"--quiet",
		)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Checkov pode retornar exit code != 0 se encontrar issues
			if cmd.ProcessState.ExitCode() > 1 {
				return nil, fmt.Errorf("checkov scan failed for %s: %w, stderr: %s",
					planFile, err, stderr.String())
			}
		}

		result.RawOutput += stdout.String()

		// ✅ parseOutput implementado abaixo
		pathResult, err := c.parseOutput(stdout.Bytes())
		if err != nil {
			return nil, fmt.Errorf("failed to parse checkov output for %s: %w", planFile, err)
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

// checkovOutput representa a estrutura JSON do Checkov
type checkovOutput struct {
	Summary struct {
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
	} `json:"summary"`
	Results struct {
		FailedChecks []struct {
			CheckID     string `json:"check_id"`
			CheckName   string `json:"check_name"`
			CheckResult struct {
				Result string `json:"result"`
			} `json:"check_result"`
			CodeBlock       [][]interface{} `json:"code_block"`
			FileAbsPath     string          `json:"file_abs_path"`
			FileLineRange   []int           `json:"file_line_range"`
			Resource        string          `json:"resource"`
			Evaluations     interface{}     `json:"evaluations"`
			CheckClass      string          `json:"check_class"`
			FixedDefinition interface{}     `json:"fixed_definition"`
			Guideline       string          `json:"guideline"`
		} `json:"failed_checks"`
		PassedChecks  []interface{} `json:"passed_checks"`
		SkippedChecks []interface{} `json:"skipped_checks"`
	} `json:"results"`
}

// ✅ parseOutput implementado
func (c *CheckovScanner) parseOutput(data []byte) (PathScan, error) {
	var output checkovOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return PathScan{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	scan := PathScan{
		Issues: []Issue{},
	}

	for _, check := range output.Results.FailedChecks {
		// Checkov não tem severity explícito, vamos inferir
		severity := inferCheckovSeverity(check.CheckClass, check.CheckID)

		issue := Issue{
			ID:           check.CheckID,
			Title:        check.CheckName,
			Severity:     severity,
			ResourceName: check.Resource,
			Description:  check.Guideline,
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

// inferCheckovSeverity infere a severidade baseado na classe do check
func inferCheckovSeverity(checkClass, checkID string) Severity {
	// Alguns checks críticos conhecidos
	criticalChecks := map[string]bool{
		"CKV_AWS_18":   true, // S3 bucket without encryption
		"CKV_AWS_19":   true, // S3 bucket without server side encryption
		"CKV_AWS_21":   true, // S3 bucket versioning not enabled
		"CKV_GCP_6":    true, // Cloud SQL database publicly accessible
		"CKV_AZURE_35": true, // Storage account allows public access
	}

	if criticalChecks[checkID] {
		return SeverityCritical
	}

	// Checks relacionados a segurança são HIGH por padrão
	checkClassLower := strings.ToLower(checkClass)
	if strings.Contains(checkClassLower, "encryption") ||
		strings.Contains(checkClassLower, "iam") ||
		strings.Contains(checkClassLower, "secret") {
		return SeverityHigh
	}

	// Outros são MEDIUM
	return SeverityMedium
}
