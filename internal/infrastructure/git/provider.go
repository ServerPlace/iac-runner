package git

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Provider implementa safety.ContentProvider usando go-git puro
type Provider struct {
	RootDir string
}

func NewProvider(rootDir string) *Provider {
	return &Provider{RootDir: rootDir}
}

// ReadAtCommit lê o conteúdo de um arquivo (blob) diretamente do histórico (Tree)
// sem precisar fazer checkout físico no disco.
func (p *Provider) ReadAtCommit(sha, relPath string) ([]byte, error) {
	// 1. Abre o repo
	repo, err := git.PlainOpen(p.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo at %s: %w", p.RootDir, err)
	}

	// 2. Resolve o Commit Object
	hash := plumbing.NewHash(sha)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("commit not found %s: %w", sha, err)
	}

	// 3. Pega a Tree (estrutura de pastas do commit)
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for commit %s: %w", sha, err)
	}

	// 4. Busca o Arquivo na Tree
	// O go-git navega na árvore virtualmente
	file, err := tree.File(relPath)
	if err != nil {
		// Retorna erro específico se arquivo não existia naquele commit
		return nil, fmt.Errorf("file %s not found in commit %s: %w", relPath, sha, err)
	}

	// 5. Lê o conteúdo do Blob
	reader, err := file.Blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to read blob: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ReadFromDisk continua usando os do Go, pois é o estado atual do workspace (Worktree)
// Poderíamos usar repo.Worktree().Filesystem.Open(relPath), mas os.ReadFile é mais direto.
func (p *Provider) ReadFromDisk(relPath string) ([]byte, error) {
	fullPath := filepath.Join(p.RootDir, relPath)
	return os.ReadFile(fullPath)
}
