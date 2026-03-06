package compliance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Helper para criar estrutura de arquivos temporária
func createStructure(t *testing.T, files []string) string {
	t.Helper()
	tmpDir := t.TempDir() // Cria diretório temporário único para o teste

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("setup: failed to create dir for %s: %v", file, err)
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("setup: failed to create file %s: %v", file, err)
		}
		f.Close()
	}
	return tmpDir
}

func TestValidateStackHierarchy(t *testing.T) {
	tests := []struct {
		name         string
		structure    []string // Arquivos a serem criados
		workDir      string   // Onde o "terragrunt.hcl" estaria (stack path)
		initialState State    // Ponto de entrada na máquina de estados (default: StateStart)
		expectError  bool
	}{
		{
			name: "Happy Path - Hierarquia Completa",
			structure: []string{
				"root.hcl",
				"org/folder.hcl",
				"org/prd/env.hcl",
				"org/prd/us-east-1/region.hcl",
				"org/prd/us-east-1/app/terragrunt.hcl",
			},
			workDir:     "org/prd/us-east-1/app",
			expectError: false,
		},
		{
			name: "Missing Root - Deve falhar",
			structure: []string{
				// "root.hcl", <-- Faltando
				"org/folder.hcl",
				"org/prd/env.hcl",
				"org/prd/us-east-1/region.hcl",
				"org/prd/us-east-1/app/terragrunt.hcl",
			},
			workDir:     "org/prd/us-east-1/app",
			expectError: true,
		},
		{
			name: "Missing Env - Transição Inválida (Region -> Folder)",
			structure: []string{
				"root.hcl",
				"org/folder.hcl",
				// "org/prd/env.hcl", <-- Faltando
				"org/prd/us-east-1/region.hcl",
				"org/prd/us-east-1/app/terragrunt.hcl",
			},
			workDir:     "org/prd/us-east-1/app",
			expectError: true,
		},
		{
			name: "Stack Profunda duplo folder - Deve subir pastas vazias até achar Region",
			structure: []string{
				"root.hcl",
				"example.com/platform/folder.hcl",
				"example.com/platform/common/folder.hcl",
				"example.com/platform/common/logging/env.hcl",
				"example.com/platform/common/logging/us-central1/region.hcl",
				"example.com/platform/common/logging/us-central1/logging/bucket/bkt-platform-logging-sink/terragrunt.hcl",
			},
			workDir:     "example.com/platform/common/logging/us-central1/logging/bucket/bkt-platform-logging-sink",
			expectError: false,
		},
		{
			name: "Stack Profunda - Deve subir pastas vazias até achar Region",
			structure: []string{
				"root.hcl",
				"org/folder.hcl",
				"org/prd/env.hcl",
				"org/prd/us-east-1/region.hcl",
				"org/prd/us-east-1/cluster-k8s/namespaces/app/terragrunt.hcl",
			},
			workDir:     "org/prd/us-east-1/cluster-k8s/namespaces/app",
			expectError: false,
		},
		{
			name: "Exception Folder - Folder GCP nao pode estar dentro de env/region",
			structure: []string{
				"root.hcl",
				"org/folder.hcl",
				"org/prd/env.hcl",
				// folder GCP deve estar diretamente sob folder.hcl, nao sob env
				"org/prd/folder/my-folder/terragrunt.hcl",
			},
			workDir:     "org/prd/folder/my-folder",
			expectError: true,
		},
		{
			name: "Org Component - StateFoundEnv pula region e env",
			structure: []string{
				"root.hcl",
				"org/folder.hcl",
				"org/prd/org-policy/terragrunt.hcl",
			},
			workDir:      "org/prd/org-policy",
			initialState: StateFoundEnv,
			expectError:  false,
		},
		{
			name: "Org Component - sem folder.hcl deve falhar mesmo com StateFoundEnv",
			structure: []string{
				"root.hcl",
				// folder.hcl ausente
				"org/prd/org-policy/terragrunt.hcl",
			},
			workDir:      "org/prd/org-policy",
			initialState: StateFoundEnv,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := createStructure(t, tt.structure)

			// Caminho absoluto para a stack que vamos testar
			targetStack := filepath.Join(rootDir, tt.workDir)

			err := ValidateStackHierarchy(context.TODO(), targetStack, tt.initialState)

			if tt.expectError && err == nil {
				t.Errorf("Esperava erro, mas não ocorreu")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Não esperava erro, mas ocorreu: %v", err)
			}
		})
	}
}
