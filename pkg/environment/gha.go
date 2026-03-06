package environment

import (
	"os"
	"strings"
)

type ghaDetector struct{}

func (*ghaDetector) Provider() Provider { return ProviderGitHubActions }

func (*ghaDetector) Detect() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func (*ghaDetector) Load() (Environment, error) {
	raw := map[string]string{}
	get := func(k string) string {
		v := os.Getenv(k)
		if v != "" {
			raw[k] = v
		}
		return v
	}

	eventName := get("GITHUB_EVENT_NAME")
	repo := get("GITHUB_REPOSITORY")
	workspace := get("GITHUB_WORKSPACE")
	sha := get("GITHUB_SHA")
	sbt := get("SOURCE_COMMIT_ID")
	user := get("GITHUB_ACTOR") // GitHub username

	env := Environment{
		Provider:      ProviderGitHubActions,
		SCM:           SCMGitHub,
		Event:         mapGHAEvent(eventName),
		RepoName:      repo,
		Workspace:     workspace,
		CheckedOutSHA: sha,
		SourceBranch:  sbt,
		User:          user,
		Raw:           raw,
	}
	return env, nil
}

func mapGHAEvent(e string) EventType {
	switch strings.ToLower(e) {
	case "pull_request", "pull_request_target":
		return EventPullRequest
	case "push":
		return EventPush
	case "workflow_dispatch":
		return EventManual
	default:
		return EventUnknown
	}
}
