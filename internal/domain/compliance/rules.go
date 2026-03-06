package compliance

import (
	"context"
	"fmt"
	"path/filepath"

	iHcl "github.com/ServerPlace/iac-runner/internal/infrastructure/hcl"
	"github.com/ServerPlace/iac-runner/internal/domain/git"
)

// StructResult representa o resultado da análise de compliance.
type StructResult struct {
	IsAdminChange   bool
	Issues          []string
	AdminFiles      []string
	TerragruntFiles map[string]bool
}

const (
	FileRoot       = "root.hcl"
	FileEnv        = "env.hcl"
	FileFolder     = "folder.hcl"
	FileRegion     = "region.hcl"
	terragruntFile = "terragrunt.hcl"
)

// StructuralFiles são os arquivos de hierarquia que não são stacks executáveis.
var StructuralFiles = []string{FileRoot, FileEnv, FileFolder, FileRegion}

// ContentProvider é a porta de entrada para leitura de conteúdo de arquivos.
type ContentProvider interface {
	ReadFromDisk(path string) ([]byte, error)
	ReadAtCommit(sha, path string) ([]byte, error)
}

// CheckArchitecture valida a arquitetura dos arquivos alterados.
// moduleRoot, quando não vazio, é o único repositório de módulos autorizado
// (ex: "https://dev.azure.com/org/proj/_git/blueprints"). Qualquer source
// com base diferente é bloqueado.
func CheckArchitecture(ctx context.Context, files []git.FileChange, r Registry, provider ContentProvider, moduleRoot, baseSHA string) (StructResult, error) {
	result := StructResult{
		Issues:          []string{},
		AdminFiles:      []string{},
		TerragruntFiles: map[string]bool{},
	}

	for _, file := range files {
		base := filepath.Base(file.Path)

		// Regra 1: Arquivos Admin (estruturais)
		if base == FileRoot || base == FileEnv || base == FileFolder {
			result.AdminFiles = append(result.AdminFiles, file.Path)
			result.IsAdminChange = true
		}
		if isProject(file.Path) || isFolder(file.Path) {
			result.AdminFiles = append(result.AdminFiles, file.Path)
			result.IsAdminChange = true
		}

		// Regra 2: Stacks terragrunt
		if base != terragruntFile {
			continue
		}

		// 2a. Ler conteúdo para parse
		// Para arquivos deletados, o arquivo não existe no disco: lê do histórico git.
		var content []byte
		var err error
		if file.ChangeType == git.ChangeDelete {
			content, err = provider.ReadAtCommit(baseSHA, file.Path)
		} else {
			content, err = provider.ReadFromDisk(file.Path)
		}
		if err != nil {
			return result, fmt.Errorf("compliance: falha ao ler %s: %w", file.Path, err)
		}

		// 2b. Parsear source do módulo
		analysis := iHcl.NewAnalisysFromBytes(content, file.Path)
		sourceRef, diags := analysis.ParseTerraformSource()
		if diags.HasErrors() {
			return result, fmt.Errorf("compliance: source inválido em %s: %s", file.Path, diags.Error())
		}
		if sourceRef == nil {
			return result, fmt.Errorf("compliance: %s não declara terraform.source", file.Path)
		}

		// 2c. Validar repositório de origem
		if moduleRoot != "" && sourceRef.Base != moduleRoot {
			return result, fmt.Errorf("compliance: repositório não autorizado em %s: esperado %q, encontrado %q",
				file.Path, moduleRoot, sourceRef.Base)
		}

		// 2d. Lookup no registry
		entry, found := r.Lookup(sourceRef.Path)
		if !found {
			return result, fmt.Errorf("compliance: módulo %q não encontrado no registry (latest.json)", sourceRef.Path)
		}

		// 2e. Validar hierarquia — ignorado para deletes (stack já existe)
		if file.ChangeType != git.ChangeDelete {
			hierarchyStart := StateStart
			if entry.Organization {
				hierarchyStart = StateFoundEnv
			}
			if err := ValidateStackHierarchy(ctx, file.Path, hierarchyStart); err != nil {
				return result, err
			}
		}

		// 2f. Validar versão do módulo
		if err := validateVersion(file.Path, sourceRef.Revision, entry.MinVersions, file.ChangeType); err != nil {
			return result, err
		}

		// 2g. Validar inputs proibidos
		diag := NewValidatorFromBytes(content, file.Path,
			WithProhibitedInputKeys("project_id", "region", "folder_id"),
		).Run()
		if diag.HasErrors() {
			return result, fmt.Errorf("compliance: inputs proibidos em %s: %s", file.Path, diag.Error())
		}

		result.TerragruntFiles[file.Path] = true
	}

	return result, nil
}

// validateVersion verifica se a versão declarada no source atende ao mínimo exigido pelo registry.
func validateVersion(path, revision string, min MinVersions, changeType git.ChangeType) error {
	declared, err := parseVersion(revision)
	if err != nil {
		return fmt.Errorf("compliance: versão inválida em %s: %w", path, err)
	}

	var required int
	switch changeType {
	case git.ChangeAdd:
		required = min.Create
	case git.ChangeModify:
		required = min.Update
	case git.ChangeDelete, git.ChangeRename:
		required = min.Destroy
	}

	if declared < required {
		return fmt.Errorf("compliance: versão %d em %s está abaixo do mínimo %d para operação %s",
			declared, path, required, changeType.String())
	}

	return nil
}

func isFolder(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "folder"
}

func isProject(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "project"
}
