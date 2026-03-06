// internal/domain/security/trivy.go
package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ServerPlace/iac-runner/pkg/command"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"os"
	"os/exec"
	"path/filepath"
)

type TrivyScanner struct {
	binPath string
}

func NewTrivyScanner(binPath string) *TrivyScanner {
	if binPath == "" {
		binPath = "trivy"
	}
	return &TrivyScanner{binPath: binPath}
}

func (t *TrivyScanner) Name() string {
	return "trivy"
}

// ScanPlanFiles recebe paths de arquivos e os renomeia temporariamente se necessário
func (t *TrivyScanner) ScanPlanFiles(ctx context.Context, planFiles []string) (*ScanResult, error) {
	logger := log.FromContext(ctx)

	result := &ScanResult{
		Scanner: t.Name(),
		ByPath:  make(map[string]PathScan),
	}

	for _, planFile := range planFiles {
		// ✅ IMPORTANTE: Trivy reconhece plan JSON apenas com extensão .tfplan.json
		// Se o arquivo não tem essa extensão, criar link simbólico temporário
		scanFile := planFile
		var tempFile string

		if !isTrivyCompatibleName(planFile) {
			tempFile = planFile + ".tfplan.json"

			// Criar link simbólico temporário
			if err := os.Symlink(planFile, tempFile); err != nil {
				// Se symlink falhar, copiar o arquivo
				data, err := os.ReadFile(planFile)
				if err != nil {
					return nil, fmt.Errorf("failed to read plan file %s: %w", planFile, err)
				}
				if err := os.WriteFile(tempFile, data, 0644); err != nil {
					return nil, fmt.Errorf("failed to create temp file: %w", err)
				}
			}

			scanFile = tempFile
			defer os.Remove(tempFile) // Limpar depois

			logger.Debug().Msgf("Created temporary file %s for Trivy scan", tempFile)
		}

		// ✅ Comando correto do Trivy
		params := []string{
			"config",
			"--format", "json",
			"--severity", "CRITICAL,HIGH,MEDIUM,LOW",
			scanFile, // ✅ Arquivo individual (não diretório)
		}

		opts := command.RunOptions{
			LiveOutput: true,
		}

		output, err := command.Run(ctx, t.binPath, params, opts)

		// Tratamento de exit code
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				if ee.ExitCode() == 1 {
					// Exit code 1 = issues encontradas, continuar
					err = nil
				} else {
					// Exit code 2+ = erro real
					return nil, fmt.Errorf("trivy scan failed for %s: %w", planFile, err)
				}
			} else {
				return nil, fmt.Errorf("trivy scan failed for %s: %w", planFile, err)
			}
		}

		result.RawOutput += output

		pathResult, err := t.parseOutput([]byte(output))
		if err != nil {
			return nil, fmt.Errorf("failed to parse trivy output for %s: %w\n\n%s", planFile, err, output)
		}

		pathResult.Path = planFile // ✅ Path original (não o temp)
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

// isTrivyCompatibleName verifica se o nome do arquivo é reconhecido pelo Trivy
func isTrivyCompatibleName(path string) bool {
	// Trivy reconhece: *.tfplan.json
	return filepath.Ext(path) == ".json" &&
		(filepath.Ext(filepath.Base(path[:len(path)-5])) == ".tfplan")
}

// trivyOutput representa a estrutura JSON do Trivy
type trivyOutput struct {
	Results []struct {
		Target            string `json:"Target"`
		Misconfigurations []struct {
			ID            string `json:"ID"`
			AVDID         string `json:"AVDID"`
			Title         string `json:"Title"`
			Description   string `json:"Description"`
			Message       string `json:"Message"`
			Severity      string `json:"Severity"`
			Resolution    string `json:"Resolution"`
			CauseMetadata struct {
				Resource  string `json:"Resource"`
				Provider  string `json:"Provider"`
				Service   string `json:"Service"`
				StartLine int    `json:"StartLine"`
				EndLine   int    `json:"EndLine"`
			} `json:"CauseMetadata"`
		} `json:"Misconfigurations"`
	} `json:"Results"`
}

func (t *TrivyScanner) parseOutput(data []byte) (PathScan, error) {
	var output trivyOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return PathScan{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	scan := PathScan{
		Issues: []Issue{},
	}

	for _, result := range output.Results {
		for _, misconfig := range result.Misconfigurations {
			severity := parseSeverity(misconfig.Severity)

			issue := Issue{
				ID:           misconfig.ID,
				Title:        misconfig.Title,
				Severity:     severity,
				Description:  misconfig.Message,
				ResourceType: misconfig.CauseMetadata.Service,
				ResourceName: misconfig.CauseMetadata.Resource,
			}

			scan.Issues = append(scan.Issues, issue)

			if severity == SeverityCritical {
				scan.HasCritical = true
			}

			if scan.Severity == "" || severity > scan.Severity {
				scan.Severity = severity
			}
		}
	}

	return scan, nil
}

func parseSeverity(s string) Severity {
	switch s {
	case "CRITICAL":
		return SeverityCritical
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityUnknown
	}
}
