// pkg/cigroup/cigroup.go
// Wraps console output in CI-specific collapsible sections.
//
// Usage:
//
//	g := cigroup.New(env.Provider)
//	g.Open("terragrunt init")
//	defer g.Close("terragrunt init")
package cigroup

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ServerPlace/iac-runner/pkg/environment"
)

// Grouper emits CI-specific open/close markers for collapsible log sections.
type Grouper interface {
	Open(name string)
	Close(name string)
}

// New returns the Grouper matching the given CI provider.
// Returns a no-op Grouper for local/unknown environments.
func New(provider environment.Provider) Grouper {
	switch provider {
	case environment.ProviderAzurePipelines:
		return &azureGrouper{}
	case environment.ProviderGitHubActions:
		return &githubGrouper{}
	case environment.ProviderGitLabCI:
		return &gitlabGrouper{}
	default:
		return &nopGrouper{}
	}
}

// --- Azure Pipelines ---

type azureGrouper struct{}

func (a *azureGrouper) Open(name string) {
	fmt.Fprintf(os.Stdout, "##[group]%s\n", name)
}

func (a *azureGrouper) Close(_ string) {
	fmt.Fprintln(os.Stdout, "##[endgroup]")
}

// --- GitHub Actions ---

type githubGrouper struct{}

func (g *githubGrouper) Open(name string) {
	fmt.Fprintf(os.Stdout, "::group::%s\n", name)
}

func (g *githubGrouper) Close(_ string) {
	fmt.Fprintln(os.Stdout, "::endgroup::")
}

// --- GitLab CI ---
// Format: \e[0Ksection_start:TIMESTAMP:section_id\r\e[0KTitle
//         \e[0Ksection_end:TIMESTAMP:section_id\r\e[0K

type gitlabGrouper struct{}

func (g *gitlabGrouper) Open(name string) {
	fmt.Fprintf(os.Stdout, "\x1b[0Ksection_start:%d:%s\r\x1b[0K%s\n",
		time.Now().Unix(), sectionID(name), name)
}

func (g *gitlabGrouper) Close(name string) {
	fmt.Fprintf(os.Stdout, "\x1b[0Ksection_end:%d:%s\r\x1b[0K\n",
		time.Now().Unix(), sectionID(name))
}

// sectionID normalizes a name into a valid GitLab section identifier (no spaces, lowercase).
func sectionID(name string) string {
	return strings.ToLower(strings.NewReplacer(" ", "_", "/", "_", ":", "_").Replace(name))
}

// --- No-op (local / unknown) ---

type nopGrouper struct{}

func (n *nopGrouper) Open(_ string)  {}
func (n *nopGrouper) Close(_ string) {}

// Nop returns a no-op Grouper that emits nothing.
// Use as a safe default when no CI provider is configured.
func Nop() Grouper { return &nopGrouper{} }
