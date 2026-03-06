package environment

import (
	"fmt"
	"os"
	"strings"
)

type gitlabDetector struct{}

func (*gitlabDetector) Provider() Provider { return ProviderGitLabCI }

func (*gitlabDetector) Detect() bool {
	return os.Getenv("GITLAB_CI") == "true"
}

func (*gitlabDetector) Load() (Environment, error) {
	raw := map[string]string{}
	get := func(k string) string {
		v := os.Getenv(k)
		if v != "" {
			raw[k] = v
		}
		return v
	}

	source := get("CI_PIPELINE_SOURCE")

	iid := get("CI_MERGE_REQUEST_IID")
	src := get("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME")
	dst := get("CI_MERGE_REQUEST_TARGET_BRANCH_NAME")

	projectPath := get("CI_PROJECT_PATH")
	projectURL := get("CI_PROJECT_URL")
	workspace := get("CI_PROJECT_DIR")
	sha := get("CI_COMMIT_SHA")
	sbt := get("CI_MERGE_REQUEST_SOURCE_BRANCH_SHA")

	user := get("GITLAB_USER_LOGIN") // Username
	if user == "" {
		user = get("GITLAB_USER_EMAIL") // Email como fallback
	}
	if user == "" {
		user = get("GITLAB_USER_NAME") // Nome completo como último recurso
	}
	env := Environment{
		Provider:        ProviderGitLabCI,
		SCM:             SCMGitLab,
		Event:           mapGitLabEvent(source, iid),
		ChangeNumber:    iid,
		SourceBranch:    src,
		TargetBranch:    dst,
		RepoName:        projectPath,
		RepoURL:         projectURL,
		Workspace:       workspace,
		CheckedOutSHA:   sha,
		SourceBranchSHA: sbt,
		User:            user,
		Raw:             raw,
	}

	if env.Event == EventMergeRequest {
		if env.ChangeNumber == "" || env.SourceBranch == "" || env.TargetBranch == "" {
			return Environment{}, fmt.Errorf("gitlab MR context incomplete: IID=%q SRC=%q DST=%q",
				env.ChangeNumber, env.SourceBranch, env.TargetBranch)
		}
	}
	if env.Workspace == "" {
		return Environment{}, fmt.Errorf("gitlab workspace missing (CI_PROJECT_DIR empty)")
	}
	return env, nil
}

func mapGitLabEvent(source, iid string) EventType {
	// Se IID existe, é MR
	if strings.TrimSpace(iid) != "" {
		return EventMergeRequest
	}
	switch source {
	case "push":
		return EventPush
	case "web", "trigger", "api", "pipeline":
		return EventManual
	default:
		return EventUnknown
	}
}
