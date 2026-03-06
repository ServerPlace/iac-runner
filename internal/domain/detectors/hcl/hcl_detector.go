package hcl

import (
	"context"
	"fmt"
	intGit "github.com/ServerPlace/iac-runner/internal/domain/git"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// HCLDetector é responsável por identificar mudanças relevantes
// em arquivos .hcl entre dois commits Git.
type HCLDetector struct{}

// NewHCLDetector cria o detector.
// Mantido simples porque não há estado interno.
func NewHCLDetector() *HCLDetector {
	return &HCLDetector{}
}

// Detect compara dois SHAs (base e head) e retorna apenas
// mudanças relevantes para arquivos .hcl.
func (d *HCLDetector) Detect(
	ctx context.Context,
	repoDir string,
	baseSHA string,
	checkedOutSHA string,
) (intGit.Target, error) {

	// Permite cancelamento imediato se o chamador desistir
	select {
	case <-ctx.Done():
		return intGit.Target{}, ctx.Err()
	default:
	}

	// Abre o repositório Git local
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return intGit.Target{}, fmt.Errorf("open repo: %w", err)
	}

	// Resolve os commits base e head a partir dos SHAs
	baseCommit, err := commitBySHA(repo, baseSHA)
	if err != nil {
		return intGit.Target{}, err
	}
	checkedOutCommit, err := commitBySHA(repo, checkedOutSHA)
	if err != nil {
		return intGit.Target{}, err
	}

	// Extrai as árvores (snapshot de arquivos) de cada commit
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return intGit.Target{}, fmt.Errorf("base tree: %w", err)
	}
	headTree, err := checkedOutCommit.Tree()
	if err != nil {
		return intGit.Target{}, fmt.Errorf("head tree: %w", err)
	}

	// Calcula o diff estrutural entre as duas árvores
	changes, err := object.DiffTree(baseTree, headTree)
	if err != nil {
		return intGit.Target{}, fmt.Errorf("diff tree: %w", err)
	}

	// Coleta final de arquivos relevantes
	files := make([]intGit.FileChange, 0, len(changes))

	// Itera por cada mudança detectada pelo Git
	for _, ch := range changes {

		// Checa cancelamento a cada iteração (seguro e barato)
		select {
		case <-ctx.Done():
			return intGit.Target{}, ctx.Err()
		default:
		}

		// Processa uma única mudança e,
		// se relevante, adiciona em files
		processChange(ch, &files)
	}

	// Resultado final do diff
	return intGit.Target{
		BaseSHA:       baseSHA,
		CheckedOutSHA: checkedOutSHA,
		Files:         files,
	}, nil
}

// commitBySHA resolve um SHA para um objeto Commit,
// encapsulando o erro com contexto.
func commitBySHA(repo *git.Repository, sha string) (*object.Commit, error) {
	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("commit %s: %w", sha, err)
	}
	return commit, nil
}

// processChange interpreta uma mudança do Git (insert, delete, modify)
// e decide se ela é relevante para arquivos .hcl.
func processChange(
	ch *object.Change,
	files *[]intGit.FileChange,
) {
	act, err := ch.Action()
	if err != nil {
		panic(fmt.Errorf("change action: %w", err))
	}

	from := cleanGitPath(ch.From.Name)
	to := cleanGitPath(ch.To.Name)

	switch act {

	case merkletrie.Insert:
		if isHCL(to) {
			appendChange(files, intGit.FileChange{
				Path:       to,
				ChangeType: intGit.ChangeAdd,
			})
		}

	case merkletrie.Delete:
		if isHCL(from) {
			appendChange(files, intGit.FileChange{
				Path:       from,
				ChangeType: intGit.ChangeDelete,
			})
		}

	case merkletrie.Modify:
		handleModify(from, to, files)
	}
}

// handleModify separa claramente:
// - rename (from != to)
// - modificação simples
func handleModify(
	from string,
	to string,
	files *[]intGit.FileChange,
) {

	if from != "" && to != "" && from != to {
		if isHCL(from) || isHCL(to) {
			appendChange(files, intGit.FileChange{
				Path:       to,
				OldPath:    from,
				ChangeType: intGit.ChangeRename,
			})
		}
		return
	}

	filePath := to
	if filePath == "" {
		filePath = from
	}

	if isHCL(filePath) {
		appendChange(files, intGit.FileChange{
			Path:       filePath,
			OldPath:    from,
			ChangeType: intGit.ChangeModify,
		})
	}
}

// cleanGitPath normaliza paths vindos do Git.
func cleanGitPath(p string) string {
	if p == "" {
		return ""
	}
	return path.Clean(p)
}

// isHCL verifica se o arquivo é relevante (.hcl).
func isHCL(p string) bool {
	return strings.HasSuffix(strings.ToLower(p), ".hcl")
}

// appendChange centraliza o append.
func appendChange(
	files *[]intGit.FileChange,
	change intGit.FileChange,
) {
	*files = append(*files, change)
}
