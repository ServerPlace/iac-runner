package compliance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	domainGit "github.com/ServerPlace/iac-runner/internal/domain/git"
)

// diskProvider satisfaz ContentProvider lendo os arquivos pelo caminho recebido.
type diskProvider struct {
	// oldContents mapeia path -> conteúdo "antigo" para simular arquivos deletados.
	oldContents map[string][]byte
}

func (d diskProvider) ReadFromDisk(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (d diskProvider) ReadAtCommit(_ string, path string) ([]byte, error) {
	if d.oldContents != nil {
		if content, ok := d.oldContents[path]; ok {
			return content, nil
		}
	}
	return nil, fmt.Errorf("ReadAtCommit: arquivo não encontrado: %s", path)
}

// testModuleRoot é a raiz de repositório usada nos testes.
const testModuleRoot = "https://github.com/test/modules.git"

// testModulePath é o subdiretório do módulo após o "//".
const testModulePath = "modules/app"

// makeTestRegistry retorna um Registry mínimo para os testes.
func makeTestRegistry() Registry {
	return Registry{
		testModulePath: ModuleEntry{
			Path:         testModulePath,
			Organization: false,
			MinVersions:  MinVersions{},
		},
	}
}

// validTerragruntHCL é um arquivo terragrunt.hcl válido para os testes.
const validTerragruntHCL = `
terraform {
  source = "git::https://github.com/test/modules.git//modules/app?ref=v1"
}
`

// writeFile escreve conteúdo em um arquivo já existente no tmpDir.
func writeFile(t *testing.T, tmpDir, relPath, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(tmpDir, relPath), []byte(content), 0644); err != nil {
		t.Fatalf("setup: failed to write %s: %v", relPath, err)
	}
}

func TestCheckArchitecture(t *testing.T) {
	type fileInput struct {
		relPath    string
		changeType domainGit.ChangeType
	}

	tests := []struct {
		name           string
		structure      []string
		fileContent    map[string]string // relPath -> conteúdo no disco (arquivos presentes)
		oldFileContent map[string]string // relPath -> conteúdo "antigo" para simular arquivos deletados
		changedFiles   []fileInput
		expectAdmin    bool
		expectIssues   bool
	}{
		{
			name:         "Mudança Admin - root.hcl",
			structure:    []string{"root.hcl"},
			changedFiles: []fileInput{{"root.hcl", domainGit.ChangeModify}},
			expectAdmin:  true,
			expectIssues: false,
		},
		{
			name:         "Mudança Admin - Pasta project",
			structure:    []string{"infra/project/main.tf"},
			changedFiles: []fileInput{{"infra/project/main.tf", domainGit.ChangeModify}},
			expectAdmin:  true,
			expectIssues: false,
		},
		{
			name: "Mudança Comum - Stack Válida",
			structure: []string{
				"root.hcl",
				"env/folder.hcl",
				"env/dev/env.hcl",
				"env/dev/us/region.hcl",
				"env/dev/us/app/terragrunt.hcl",
			},
			fileContent: map[string]string{
				"env/dev/us/app/terragrunt.hcl": validTerragruntHCL,
			},
			changedFiles: []fileInput{{"env/dev/us/app/terragrunt.hcl", domainGit.ChangeAdd}},
			expectAdmin:  false,
			expectIssues: false,
		},
		{
			name: "Mudança Comum - Stack Inválida (Topologia Quebrada)",
			structure: []string{
				// Faltando root.hcl e folder.hcl
				"env/dev/env.hcl",
				"env/dev/us/region.hcl",
				"env/dev/us/app/terragrunt.hcl",
			},
			fileContent: map[string]string{
				"env/dev/us/app/terragrunt.hcl": validTerragruntHCL,
			},
			changedFiles: []fileInput{{"env/dev/us/app/terragrunt.hcl", domainGit.ChangeAdd}},
			expectAdmin:  false,
			expectIssues: true,
		},
		{
			name:         "Arquivo Ignorado",
			structure:    []string{"README.md"},
			changedFiles: []fileInput{{"README.md", domainGit.ChangeModify}},
			expectAdmin:  false,
			expectIssues: false,
		},
		{
			name: "Delete - Stack válida lida do histórico git",
			structure: []string{
				"root.hcl",
				"env/folder.hcl",
				"env/dev/env.hcl",
				"env/dev/us/region.hcl",
				// terragrunt.hcl não está no disco (foi deletado)
			},
			oldFileContent: map[string]string{
				"env/dev/us/app/terragrunt.hcl": validTerragruntHCL,
			},
			changedFiles: []fileInput{{"env/dev/us/app/terragrunt.hcl", domainGit.ChangeDelete}},
			expectAdmin:  false,
			expectIssues: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := createStructure(t, tt.structure)

			for relPath, content := range tt.fileContent {
				writeFile(t, rootDir, relPath, content)
			}

			// Monta oldContents com caminhos absolutos para o diskProvider.
			oldContents := map[string][]byte{}
			for relPath, content := range tt.oldFileContent {
				abs := filepath.Join(rootDir, relPath)
				oldContents[abs] = []byte(content)
			}

			var files []domainGit.FileChange
			for _, cf := range tt.changedFiles {
				abs := filepath.Join(rootDir, cf.relPath)
				files = append(files, domainGit.FileChange{
					Path:       abs,
					OldPath:    abs,
					ChangeType: cf.changeType,
				})
			}

			provider := diskProvider{oldContents: oldContents}
			result, err := CheckArchitecture(context.TODO(), files, makeTestRegistry(), provider, testModuleRoot, "test-sha")

			if tt.expectIssues && err == nil {
				t.Errorf("Esperava erro de validação, recebeu nil")
			}
			if !tt.expectIssues && err != nil {
				t.Errorf("Erro inesperado: %v", err)
			}
			if result.IsAdminChange != tt.expectAdmin {
				t.Errorf("IsAdminChange: esperado %v, recebido %v", tt.expectAdmin, result.IsAdminChange)
			}
		})
	}
}

// TestHelpers — testes unitários das funções helper isFolder/isProject
func TestHelpers(t *testing.T) {
	t.Run("isProject", func(t *testing.T) {
		if !isProject("/path/to/project/file.tf") {
			t.Error("Deveria detectar pasta 'project'")
		}
		if isProject("/path/to/app/file.tf") {
			t.Error("Não deveria detectar pasta 'app'")
		}
	})

	t.Run("isFolder", func(t *testing.T) {
		if !isFolder("/path/to/folder/file.hcl") {
			t.Error("Deveria detectar pasta 'folder'")
		}
		if isFolder("/path/to/other/file.hcl") {
			t.Error("Não deveria detectar pasta 'other'")
		}
	})
}
