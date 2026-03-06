package prepare

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ServerPlace/iac-runner/internal/domain/git"
	"github.com/ServerPlace/iac-runner/internal/infrastructure/hcl"
)

type Service struct {
	RootDir string
	BaseRef string // ex: HEAD^
}

func New(rootDir, baseRef string) *Service {
	return &Service{RootDir: rootDir, BaseRef: baseRef}
}

// BuildExecutionQueue gera as listas ordenadas de Destroy e Apply
func (s *Service) BuildExecutionQueue(targets *git.Target) ([]string, []string, error) {
	var destroyList []string
	var applyList []string
	processedDirs := make(map[string]bool)

	// 1. Fase Destroy (Deletes/Renames)
	for _, f := range targets.Files {
		if !strings.HasSuffix(f.Path, "/terragrunt.hcl") {
			continue
		}
		if f.ChangeType == git.ChangeDelete || f.ChangeType == git.ChangeRename {
			dir := filepath.Dir(f.Path)
			if processedDirs[dir] {
				continue
			}

			// RESSURREIÇÃO
			if err := s.resurrectDirectory(dir); err != nil {
				return nil, nil, fmt.Errorf("falha na ressurreição de %s: %w", dir, err)
			}
			destroyList = append(destroyList, dir)
			processedDirs[dir] = true
		}
	}

	// 2. Fase Apply (Adds/Modifies/Renames)
	for _, f := range targets.Files {
		if !strings.HasSuffix(f.Path, "/terragrunt.hcl") {
			continue
		}
		if f.ChangeType == git.ChangeAdd || f.ChangeType == git.ChangeModify || f.ChangeType == git.ChangeRename {
			dir := filepath.Dir(f.Path)
			// Evita duplicatas
			isDuplicate := false
			for _, existing := range applyList {
				if existing == dir {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				applyList = append(applyList, dir)
			}
		}
	}

	// 3. Ordenação Topológica
	// True = Invertido (Destroy)
	orderedDestroy, err := s.resolveDependencies(destroyList, true)
	if err != nil {
		return nil, nil, err
	}

	// False = Normal (Apply)
	orderedApply, err := s.resolveDependencies(applyList, false)
	if err != nil {
		return nil, nil, err
	}

	return orderedDestroy, orderedApply, nil
}

func (s *Service) resurrectDirectory(relPath string) error {
	fullPath := filepath.Join(s.RootDir, relPath)
	_ = os.MkdirAll(fullPath, 0755)

	cmd := exec.Command("git", "checkout", s.BaseRef, "--", relPath)
	cmd.Dir = s.RootDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout output: %s | err: %w", string(out), err)
	}
	return nil
}

func (s *Service) resolveDependencies(dirs []string, invert bool) ([]string, error) {
	if len(dirs) == 0 {
		return []string{}, nil
	}

	graph := make(map[string][]string)
	dirSet := make(map[string]bool)
	for _, d := range dirs {
		dirSet[d] = true
	}

	parser := &hcl.Parser{}

	for _, dir := range dirs {
		hclPath := filepath.Join(s.RootDir, dir, "terragrunt.hcl")
		if _, err := os.Stat(hclPath); os.IsNotExist(err) {
			continue
		}

		mod, err := parser.ParseModule(hclPath)
		if err != nil {
			return nil, err
		}

		for _, dep := range mod.Dependencies {
			relDep, _ := filepath.Rel(s.RootDir, dep)
			if dirSet[relDep] {
				graph[dir] = append(graph[dir], relDep)
			}
		}
	}

	sorted, err := topologicalSort(graph, dirs)
	if err != nil {
		return nil, err
	}

	if invert {
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}
	}
	return sorted, nil
}

func topologicalSort(deps map[string][]string, nodes []string) ([]string, error) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []string

	var visit func(n string) error
	visit = func(n string) error {
		if visiting[n] {
			return fmt.Errorf("ciclo detectado em %s", n)
		}
		if visited[n] {
			return nil
		}
		visiting[n] = true
		for _, dep := range deps[n] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[n] = false
		visited[n] = true
		result = append(result, n)
		return nil
	}

	sort.Strings(nodes)
	for _, n := range nodes {
		if !visited[n] {
			if err := visit(n); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
