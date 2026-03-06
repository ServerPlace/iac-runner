package safety

import (
	"errors"
	"strings"
	"testing"

	"github.com/ServerPlace/iac-runner/internal/domain/git"
)

// MockContentProvider auxilia na simulação de leitura de arquivos e git
type MockContentProvider struct {
	FilesOnDisk   map[string][]byte
	FilesAtCommit map[string][]byte
	ErrorToReturn error
}

func (m *MockContentProvider) ReadAtCommit(sha, path string) ([]byte, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	content, ok := m.FilesAtCommit[path]
	if !ok {
		return nil, errors.New("file not found in git")
	}
	return content, nil
}

func (m *MockContentProvider) ReadFromDisk(path string) ([]byte, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	content, ok := m.FilesOnDisk[path]
	if !ok {
		return nil, errors.New("file not found on disk")
	}
	return content, nil
}

func TestValidateChanges(t *testing.T) {
	tests := []struct {
		name         string
		targets      *git.Target
		mockDisk     map[string][]byte
		mockGit      map[string][]byte
		wantErr      bool
		errSubstring string
	}{
		{
			name: "Success: New valid HCL file",
			targets: &git.Target{
				Files: []git.FileChange{
					{Path: "terraform.hcl", ChangeType: git.ChangeAdd},
				},
			},
			mockDisk: map[string][]byte{
				"terraform.hcl": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=10" }`),
			},
			wantErr: false,
		},
		{
			name: "Error: Syntax error in new file",
			targets: &git.Target{
				Files: []git.FileChange{
					{Path: "bad.tf", ChangeType: git.ChangeAdd},
				},
			},
			mockDisk: map[string][]byte{
				"bad.tf": []byte(`inputs {`), // Faltando fechamento
			},
			wantErr:      true,
			errSubstring: "Malformed HCL syntax",
		},
		{
			name: "Success: Modify file without changing source",
			targets: &git.Target{
				BaseSHA: "old-sha",
				Files: []git.FileChange{
					{Path: "terragrunt.hcl", OldPath: "terragrunt.hcl", ChangeType: git.ChangeModify},
				},
			},
			mockDisk: map[string][]byte{
				"terragrunt.hcl": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=10" }
`),
			},
			mockGit: map[string][]byte{
				"terragrunt.hcl": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=10" }
`),
			},
			wantErr: false,
		}, {
			name: "Success: Modify file without changing source, only ref",
			targets: &git.Target{
				BaseSHA: "old-sha",
				Files: []git.FileChange{
					{Path: "mod.tf", OldPath: "mod.tf", ChangeType: git.ChangeModify},
				},
			},
			mockDisk: map[string][]byte{
				"mod.tf": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=1"}`),
			},
			mockGit: map[string][]byte{
				"mod.tf": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=2"}`),
			},
			wantErr: false,
		},
		{
			name: "Error: Critical Swap Detected",
			targets: &git.Target{
				BaseSHA: "old-sha",
				Files: []git.FileChange{
					{Path: "mod.tf", OldPath: "mod.tf", ChangeType: git.ChangeModify},
				},
			},
			mockDisk: map[string][]byte{
				"mod.tf": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//7-integration/pubsub/?ref=2"}`),
			},
			mockGit: map[string][]byte{
				"mod.tf": []byte(`terraform { source = "git::https://dev.azure.com/example-organization/Pipeline-IAC/_git/blueprints//6-integration/pubsub/?ref=2"}`),
			},
			wantErr:      true,
			errSubstring: "CRITICAL SWAP DETECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MockContentProvider{
				FilesOnDisk:   tt.mockDisk,
				FilesAtCommit: tt.mockGit,
			}

			err := ValidateChanges(tt.targets, provider, "workspace")

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateChanges() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstring) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstring)
			}
		})
	}
}
