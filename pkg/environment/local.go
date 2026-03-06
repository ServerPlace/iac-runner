package environment

import "os"

type localDetector struct{}

func (*localDetector) Provider() Provider { return ProviderLocal }
func (*localDetector) Detect() bool       { return true }

func (*localDetector) Load() (Environment, error) {
	wd, _ := os.Getwd()
	return Environment{
		Provider:  ProviderLocal,
		SCM:       SCMUnknown,
		Event:     EventManual,
		Workspace: wd,
	}, nil
}
