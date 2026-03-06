package safety

import (
	"fmt"

	"github.com/ServerPlace/iac-runner/internal/domain/git"
	"github.com/ServerPlace/iac-runner/internal/infrastructure/hcl"
)

// ContentProvider é a porta de saída (interface) que a Infra deve implementar
type ContentProvider interface {
	ReadAtCommit(sha, path string) ([]byte, error)
	ReadFromDisk(path string) ([]byte, error)
}

// ValidateChanges verifica sintaxe HCL e detecta swaps de source.
// Pré-condição para o compliance: garante que o HCL é parseável antes de qualquer análise de conteúdo.
func ValidateChanges(targets *git.Target, provider ContentProvider, workspace string) error {
	for _, file := range targets.Files {

		// Regra 1: Validação de Sintaxe (HCL Válido)
		if file.ChangeType == git.ChangeAdd || file.ChangeType == git.ChangeModify {
			content, err := provider.ReadFromDisk(file.Path)
			if err != nil {
				return fmt.Errorf("safety: failed to read %s from disk: %w", file.Path, err)
			}

			parser := &hcl.Parser{}
			if _, err := parser.ParseBytes(content); err != nil {
				return fmt.Errorf("❌ Malformed HCL syntax in %s: %w", file.Path, err)
			}
		}

		// Regra 2: Component Swap (Troca de Source)
		if file.ChangeType == git.ChangeModify {
			oldBytes, err := provider.ReadAtCommit(targets.BaseSHA, file.OldPath)
			if err != nil {
				return fmt.Errorf("safety: failed to read base version of %s: %w", file.OldPath, err)
			}

			newBytes, err := provider.ReadFromDisk(file.Path)
			if err != nil {
				return err
			}

			if err := checkSwap(file.Path, oldBytes, newBytes); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkSwap(path string, oldContent, newContent []byte) error {
	parser := hcl.NewAnalisysFromBytes(oldContent, path)
	oa, err := parser.ParseTerraformSource()
	if oa == nil && len(err) < 1 {
		return nil
	}
	if err != nil {
		return err
	}
	na, err := hcl.NewAnalisysFromBytes(newContent, path).ParseTerraformSource()
	if err != nil {
		return err
	}
	if oa.Path != na.Path {
		return fmt.Errorf("CRITICAL SWAP DETECTED in %s:\n   Old Source: %s\n   New Source: %s\n   Action: BLOCKED. Destroy/Recreate via source change is forbidden.", path, oa.Path, na.Path)
	}
	return nil
}
