// internal/domain/plans/analysis.go
package plans

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ServerPlace/iac-runner/internal/domain/git"
	tfjson "github.com/hashicorp/terraform-json"
)

// PlanOutput representa o output de um terraform plan
type PlanOutput struct {
	Path         string
	Output       string // Output textual do plan
	PlanFilePath string // ✅ Path do plan JSON no disco
	JSONPlan     []byte // ✅ Plan em formato JSON (para análise)
}

// PlanAnalysis é o resultado da análise de múltiplos plans
type PlanAnalysis struct {
	Components []Component
}

// Component representa uma stack analisada
type Component struct {
	Path          string
	ChangeType    git.ChangeType // tipo de mudança Git
	IsReplacement bool           // se há recursos sendo replaced
	PlanFilePath  string         // ✅ Path do plan JSON (para scanners)
}

// Analyse analisa os outputs de plan e correlaciona com mudanças Git
func Analyse(plans []PlanOutput, targets *git.Target) PlanAnalysis {
	// Mapear mudanças Git por path
	changeMap := make(map[string]git.FileChange)
	for _, f := range targets.Files {
		cleanPath := strings.TrimSuffix(f.Path, "/terragrunt.hcl")
		changeMap[cleanPath] = f
	}

	analysis := PlanAnalysis{
		Components: make([]Component, 0, len(plans)),
	}

	for _, p := range plans {
		c := Component{
			Path:         p.Path,
			PlanFilePath: p.PlanFilePath, // ✅ Propagar path
		}

		// Correlacionar com mudança Git
		if fc, ok := changeMap[p.Path]; ok {
			c.ChangeType = fc.ChangeType
		}

		// Detectar replacements usando JSONPlan em memória (rápido)
		if len(p.JSONPlan) > 0 {
			var tfPlan tfjson.Plan
			if err := json.Unmarshal(p.JSONPlan, &tfPlan); err == nil {
				for _, rc := range tfPlan.ResourceChanges {
					if rc.Change != nil && rc.Change.Actions.Replace() {
						c.IsReplacement = true
						break
					}
				}
			}
		}

		analysis.Components = append(analysis.Components, c)
	}

	return analysis
}

// GenerateSummary gera um resumo textual da análise
func (a *PlanAnalysis) GenerateSummary() string {
	if len(a.Components) == 0 {
		return "📊 Plan Summary: No changes detected\n"
	}

	var summary strings.Builder
	summary.WriteString("\n📊 Plan Summary\n")
	summary.WriteString("===============\n\n")

	// Contadores
	var adds, modifies, deletes, replacements int

	for _, comp := range a.Components {
		switch comp.ChangeType {
		case git.ChangeAdd:
			adds++
		case git.ChangeModify:
			modifies++
		case git.ChangeDelete:
			deletes++
		}

		if comp.IsReplacement {
			replacements++
		}
	}

	summary.WriteString(fmt.Sprintf("Total Stacks: %d\n", len(a.Components)))
	summary.WriteString(fmt.Sprintf("├─ New:      %d\n", adds))
	summary.WriteString(fmt.Sprintf("├─ Modified: %d\n", modifies))
	summary.WriteString(fmt.Sprintf("├─ Deleted:  %d\n", deletes))
	summary.WriteString(fmt.Sprintf("└─ With Replacements: %d\n\n", replacements))

	if replacements > 0 {
		summary.WriteString("⚠️ **ALERTA**: Esta operação contém REPLACEMENT(s)!\n\n")
	}

	summary.WriteString("Details:\n")
	for _, comp := range a.Components {
		label := "📝"
		switch comp.ChangeType {
		case git.ChangeAdd:
			label = "🟢 Para Criar"
		case git.ChangeModify:
			label = "🟡 Para Alterar"
		case git.ChangeDelete:
			label = "🔴 Para Destruir"
		case git.ChangeRename:
			label = "🔀 Para Mover"
		}

		if comp.IsReplacement {
			summary.WriteString(fmt.Sprintf("  ⚠️ %s %s (REPLACEMENT)\n", label, comp.Path))
		} else {
			summary.WriteString(fmt.Sprintf("  %s %s\n", label, comp.Path))
		}
	}

	return summary.String()
}

// HasReplacements verifica se algum componente tem replacements
func (a *PlanAnalysis) HasReplacements() bool {
	for _, comp := range a.Components {
		if comp.IsReplacement {
			return true
		}
	}
	return false
}

// GetReplacementCount retorna o número de componentes com replacements
func (a *PlanAnalysis) GetReplacementCount() int {
	count := 0
	for _, comp := range a.Components {
		if comp.IsReplacement {
			count++
		}
	}
	return count
}

// GetComponentsByChangeType retorna componentes filtrados por tipo de mudança
func (a *PlanAnalysis) GetComponentsByChangeType(changeType git.ChangeType) []Component {
	var filtered []Component
	for _, comp := range a.Components {
		if comp.ChangeType == changeType {
			filtered = append(filtered, comp)
		}
	}
	return filtered
}
