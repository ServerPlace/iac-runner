package hcl

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ServerPlace/iac-runner/internal/domain/git"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHCLDetector_Integration(t *testing.T) {
	// 1. Criar diretório temporário para o repositório
	repoDir := t.TempDir()

	// 2. Inicializar um repositório Git real no disco
	repo, err := gogit.PlainInit(repoDir, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	// --- ESTADO INICIAL (Commit Base) ---
	filesBase := map[string]string{
		"keep.hcl":      "content",
		"modify_me.hcl": "original_content",
		"delete_me.hcl": "content",
	}

	for name, content := range filesBase {
		path := filepath.Join(repoDir, name)
		err = os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
		_, err = w.Add(name)
		require.NoError(t, err)
	}

	baseSHA, err := w.Commit("base commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com"},
	})
	require.NoError(t, err)

	// --- MUDANÇAS (Commit Atual) ---

	// A) Adicionar novo
	os.WriteFile(filepath.Join(repoDir, "new_file.hcl"), []byte("new"), 0644)
	w.Add("new_file.hcl")

	// B) Modificar existente
	os.WriteFile(filepath.Join(repoDir, "modify_me.hcl"), []byte("updated_content"), 0644)
	w.Add("modify_me.hcl")

	// C) Deletar
	w.Remove("delete_me.hcl")

	// D) Ignorar (Não HCL)
	os.WriteFile(filepath.Join(repoDir, "script.sh"), []byte("echo 1"), 0644)
	w.Add("script.sh")

	checkedOutSHA, err := w.Commit("apply changes", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com"},
	})
	require.NoError(t, err)

	// 3. Executar o Detector
	detector := NewHCLDetector()
	ctx := context.Background()

	result, err := detector.Detect(ctx, repoDir, baseSHA.String(), checkedOutSHA.String())
	require.NoError(t, err)

	// 4. Validações
	t.Run("Validar detecção de arquivos", func(t *testing.T) {
		// Esperamos 3 mudanças: new_file (A), modify_me (M), delete_me (D)
		// script.sh deve ser ignorado.
		assert.Equal(t, 3, len(result.Files))

		changesMap := make(map[string]git.FileChange)
		for _, f := range result.Files {
			changesMap[f.Path] = f
		}

		// Verificando o ADD
		assert.Equal(t, git.ChangeAdd, changesMap["new_file.hcl"].ChangeType)

		// Verificando o MODIFY (O ponto crítico do OldPath)
		mod := changesMap["modify_me.hcl"]
		assert.Equal(t, git.ChangeModify, mod.ChangeType)
		assert.Equal(t, "modify_me.hcl", mod.Path)
		assert.Equal(t, "modify_me.hcl", mod.OldPath, "OldPath deve ser igual ao Path em modificações")

		// Verificando o DELETE
		assert.Equal(t, git.ChangeDelete, changesMap["delete_me.hcl"].ChangeType)
	})
}
