package environment

import (
	"fmt"
	"os"
	"strings"
)

type azureDetector struct{}

func (*azureDetector) Provider() Provider { return ProviderAzurePipelines }

func (*azureDetector) Detect() bool {
	// Azure define TF_BUILD=true
	v := os.Getenv("TF_BUILD")
	return v == "True" || v == "true"
}

func (*azureDetector) Load() (Environment, error) {
	raw := map[string]string{}
	get := func(k string) string {
		v := os.Getenv(k)
		if v != "" {
			raw[k] = v
		}
		return v
	}

	buildReason := get("BUILD_REASON")
	repoProvider := get("BUILD_REPOSITORY_PROVIDER")
	repoName := get("BUILD_REPOSITORY_URI")
	workspace := get("BUILD_SOURCESDIRECTORY")
	sha := get("BUILD_SOURCEVERSION")
	sbt := get("SYSTEM_PULLREQUEST_SOURCECOMMITID")
	srcRef := get("SYSTEM_PULLREQUEST_SOURCEBRANCH")
	dstRef := get("SYSTEM_PULLREQUEST_TARGETBRANCH")
	// Tenta ler o NUMBER (Padrão GitHub)
	prNum := get("SYSTEM_PULLREQUEST_PULLREQUESTNUMBER")

	// Se estiver vazio, tenta ler o ID (Padrão Azure Repos Nativo)
	if prNum == "" {
		prNum = get("SYSTEM_PULLREQUEST_PULLREQUESTID")
	}
	// User info - Azure DevOps
	user := get("BUILD_REQUESTEDFOR") // Nome completo do user
	if user == "" {
		user = get("BUILD_REQUESTEDFOREMAIL") // Email como fallback
	}
	if user == "" {
		user = get("BUILD_REQUESTEDFORID") // ID como último recurso
	}

	env := Environment{
		Provider:        ProviderAzurePipelines,
		SCM:             mapAzureSCM(strings.ToLower(repoProvider)),
		Event:           mapAzureEvent(buildReason),
		ChangeNumber:    prNum,
		SourceRef:       srcRef,
		TargetRef:       dstRef,
		SourceBranch:    trimHeads(srcRef),
		TargetBranch:    trimHeads(dstRef),
		RepoName:        repoName,
		RepoProvider:    repoProvider,
		Workspace:       workspace,
		CheckedOutSHA:   sha,
		SourceBranchSHA: sbt,
		User:            user, // NOVO
		Raw:             raw,
	}

	// validação mínima para PR
	if env.Event == EventPullRequest {
		if env.ChangeNumber == "" || env.SourceRef == "" || env.TargetRef == "" {
			return Environment{}, fmt.Errorf("azure PR context incomplete: PR=%q SRC=%q DST=%q",
				env.ChangeNumber, env.SourceRef, env.TargetRef)
		}
	}
	// workspace quase sempre deve existir
	if env.Workspace == "" {
		return Environment{}, fmt.Errorf("azure workspace missing (BUILD_SOURCESDIRECTORY empty)")
	}

	return env, nil
}

func mapAzureEvent(reason string) EventType {
	switch reason {
	case "PullRequest":
		return EventPullRequest
	case "IndividualCI", "BatchedCI":
		return EventPush
	case "Manual":
		return EventManual
	default:
		return EventUnknown
	}
}

func mapAzureSCM(provider string) SCM {
	switch provider {
	case "github", "githubenterprise":
		return SCMGitHub
	case "tfsgit", "azuredevops":
		return SCMAzure
	default:
		return SCMUnknown
	}
}

func trimHeads(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}
