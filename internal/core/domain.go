package core

type Module struct {
	Path         string
	Dependencies []string
}

type ChangeDetector interface {
	GetChangedFiles(baseBranch string) ([]string, error)
}

type ModuleParser interface {
	ParseModule(path string) (*Module, error)
}
