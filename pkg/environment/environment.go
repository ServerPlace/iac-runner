package environment

type Provider string
type SCM string
type EventType string

const (
	ProviderAzurePipelines Provider = "azure_pipelines"
	ProviderGitLabCI       Provider = "gitlab_ci"
	ProviderGitHubActions  Provider = "github_actions"
	ProviderLocal          Provider = "local"
)

const (
	SCMGitHub  SCM = "github"
	SCMGitLab  SCM = "gitlab"
	SCMAzure   SCM = "azure_repos"
	SCMUnknown SCM = "unknown"
)

const (
	EventPullRequest  EventType = "pull_request"
	EventMergeRequest EventType = "merge_request"
	EventPush         EventType = "push"
	EventManual       EventType = "manual"
	EventUnknown      EventType = "unknown"
)

// Environment é o contrato mínimo para o executor.
// Não depende de git e não chama API externa.
type Environment struct {
	Provider Provider
	SCM      SCM
	Event    EventType

	// PR/MR
	ChangeNumber string // PR number (GitHub/Azure), MR IID (GitLab)

	// Branches (normalizados, sem refs/heads/)
	SourceBranch string
	TargetBranch string

	// Refs (quando existir)
	SourceRef string
	TargetRef string

	// Repo
	RepoName     string // ex.: org/repo (quando disponível)
	RepoProvider string // string bruta do CI
	RepoURL      string // quando disponível

	// Exec paths
	Workspace string // onde o CI checou o código

	// Commits “do CI” (pode ser merge commit no Azure PR build)
	CheckedOutSHA   string
	SourceBranchSHA string

	// Usuario em uso
	User string // NOVO

	// Debug
	Raw map[string]string
}
