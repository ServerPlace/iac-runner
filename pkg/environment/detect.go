package environment

import "fmt"

type Detector interface {
	Provider() Provider
	Detect() bool
	Load() (Environment, error)
}

func Setup() (Environment, error) {
	detectors := []Detector{
		&azureDetector{},
		&gitlabDetector{},
		&ghaDetector{},
		&localDetector{},
	}
	for _, d := range detectors {
		if d.Detect() {

			return d.Load()
		}
	}
	return Environment{}, fmt.Errorf("no CI environment detected")
}
