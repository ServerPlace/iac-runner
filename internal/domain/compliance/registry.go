package compliance

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MinVersions define a versão mínima do módulo por tipo de operação.
type MinVersions struct {
	Create  int
	Update  int
	Destroy int
}

// ModuleEntry representa um módulo registrado no latest.json.
type ModuleEntry struct {
	Path         string
	Organization bool
	MinVersions  MinVersions
}

// Registry mapeia o path do módulo à sua entrada.
type Registry map[string]ModuleEntry

// rawEntry reflete a estrutura do latest.json para desserialização.
type rawEntry struct {
	Path    string `json:"path"`
	Content struct {
		Organization bool `json:"organization"`
		MinVersions  struct {
			Create  string `json:"create"`
			Update  string `json:"update"`
			Destroy string `json:"destroy"`
		} `json:"min_versions"`
	} `json:"content"`
}

// LoadRegistry desserializa o conteúdo do latest.json em um Registry.
func LoadRegistry(data []byte) (Registry, error) {
	var raw []rawEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("registry: falha ao parsear latest.json: %w", err)
	}

	r := make(Registry, len(raw))
	for _, e := range raw {
		create, err := parseVersion(e.Content.MinVersions.Create)
		if err != nil {
			return nil, fmt.Errorf("registry: min_versions.create inválido em %q: %w", e.Path, err)
		}
		update, err := parseVersion(e.Content.MinVersions.Update)
		if err != nil {
			return nil, fmt.Errorf("registry: min_versions.update inválido em %q: %w", e.Path, err)
		}
		destroy, err := parseVersion(e.Content.MinVersions.Destroy)
		if err != nil {
			return nil, fmt.Errorf("registry: min_versions.destroy inválido em %q: %w", e.Path, err)
		}

		r[e.Path] = ModuleEntry{
			Path:         e.Path,
			Organization: e.Content.Organization,
			MinVersions: MinVersions{
				Create:  create,
				Update:  update,
				Destroy: destroy,
			},
		}
	}

	return r, nil
}

// Lookup retorna a entrada do módulo pelo path.
func (r Registry) Lookup(modulePath string) (ModuleEntry, bool) {
	entry, ok := r[modulePath]
	return entry, ok
}

// parseVersion converte uma string de versão ("24", "v24") em int.
func parseVersion(s string) (int, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("versão %q não é um inteiro válido", s)
	}
	return v, nil
}
