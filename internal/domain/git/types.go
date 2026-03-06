package git

import (
	"context"
	"encoding/json"
)

type ChangeType string

const (
	ChangeAdd    ChangeType = "A"
	ChangeModify ChangeType = "M"
	ChangeDelete ChangeType = "D"
	ChangeRename ChangeType = "R"
)

type FileChange struct {
	Path       string
	OldPath    string
	ChangeType ChangeType
}

type Target struct {
	BaseSHA       string
	CheckedOutSHA string
	SourceSHA     string
	Files         []FileChange
}
type ChangeDetector interface {
	Detect(ctx context.Context, repoDir, baseSHA, CheckedOutSHA string) (Target, error)
}

func (t *Target) Json() []byte {
	b, _ := json.MarshalIndent(t, "", " ")
	return b
}

func (c ChangeType) String() string {
	switch c {
	case ChangeAdd:
		return "Add"
	case ChangeModify:
		return "Modify"
	case ChangeDelete:
		return "Destroy"
	case ChangeRename:
		return "Move"
	default:
		return "Unknown"
	}
}
