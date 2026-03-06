package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func ResolveBranchTip(repoDir, branchName string) (string, error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	// 1. Tenta resolver localmente (ex: "master")
	hash, err := repo.ResolveRevision(plumbing.Revision(branchName))
	if err == nil {
		return hash.String(), nil
	}

	// 2. Se falhar, tenta resolver no remoto (ex: "origin/master")
	// Isso é padrão em ambientes de CI como Azure DevOps e GitHub Actions
	remoteBranch := "origin/" + branchName
	hash, err = repo.ResolveRevision(plumbing.Revision(remoteBranch))
	if err == nil {
		return hash.String(), nil
	}

	// 3. Se falhar nos dois, retorna o erro original
	return "", fmt.Errorf("resolve branch %s (and %s): %w", branchName, remoteBranch, err)
}

func ListLocalBranches(repoDir string) ([]string, error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	// repo.Branches() retorna um iterador apenas para refs/heads/*
	branchIter, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	var branches []string

	// Iteramos sobre as referências
	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		// ref.Name().Short() remove o prefixo "refs/heads/"
		// Se quiser o nome completo, use ref.Name().String()
		branches = append(branches, ref.Name().Short())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return branches, nil
}
