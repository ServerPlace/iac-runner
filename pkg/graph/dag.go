package graph

type DependencyMap map[string][]string

func (g DependencyMap) AddDependency(provider, consumer string) {
	g[provider] = append(g[provider], consumer)
}

func CalculateAffected(changedModules []string, graph DependencyMap) []string {
	affectedSet := make(map[string]bool)
	var walk func(string)

	walk = func(mod string) {
		if affectedSet[mod] {
			return
		}
		affectedSet[mod] = true
		for _, dependent := range graph[mod] {
			walk(dependent)
		}
	}

	for _, mod := range changedModules {
		walk(mod)
	}

	result := make([]string, 0, len(affectedSet))
	for mod := range affectedSet {
		result = append(result, mod)
	}
	return result
}
