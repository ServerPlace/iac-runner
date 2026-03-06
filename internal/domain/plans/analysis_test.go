package plans

import (
	"encoding/json"
	"github.com/ServerPlace/iac-runner/internal/domain/git"
	tfjson "github.com/hashicorp/terraform-json"
	"strings"
	"testing"
)

func TestGenerateSummary(t *testing.T) {
	tests := []struct {
		name      string
		analysis  PlanAnalysis
		mustHave  []string
		mustAvoid []string
	}{
		{
			name: "Relatório com Alerta de Perigo",
			analysis: PlanAnalysis{
				Components: []Component{
					{Path: "instancia-prod", ChangeType: git.ChangeModify, IsReplacement: true},
				},
			},
			mustHave: []string{"⚠️ **ALERTA**", "REPLACEMENT", "🟡 Para Alterar"},
		},
		{
			name: "Relatório Limpo",
			analysis: PlanAnalysis{
				Components: []Component{
					{Path: "new-s3", ChangeType: git.ChangeAdd, IsReplacement: false},
				},
			},
			mustHave:  []string{"🟢 Para Criar", "new-s3"},
			mustAvoid: []string{"ALERTA", "REPLACEMENT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.GenerateSummary()

			for _, s := range tt.mustHave {
				if !strings.Contains(got, s) {
					t.Errorf("[%s] Sumário deveria conter: %s\nRecebido: %s", tt.name, s, got)
				}
			}

			for _, s := range tt.mustAvoid {
				if strings.Contains(got, s) {
					t.Errorf("[%s] Sumário NÃO deveria conter: %s", tt.name, s)
				}
			}
		})
	}
}

func TestAnalyse(t *testing.T) {
	// 1. Setup do que o Git encontrou
	targets := &git.Target{
		Files: []git.FileChange{
			{Path: "stacks/network/terragrunt.hcl", ChangeType: git.ChangeAdd},
			{Path: "stacks/database/terragrunt.hcl", ChangeType: git.ChangeModify},
			{Path: "stacks/old-app/terragrunt.hcl", ChangeType: git.ChangeRename},
		},
	}

	// 2. Setup dos outputs do Terragrunt
	collected := []PlanOutput{
		{
			Path:     "stacks/network",
			JSONPlan: mockTFPlanJSON("aws_vpc.main", tfjson.Actions{tfjson.ActionCreate}),
		},
		{
			Path: "stacks/database",
			// Simula um REPLACE: O Git diz 'Modify', mas o Terraform diz 'Delete' + 'Create'
			JSONPlan: mockTFPlanJSON("aws_db.instance", tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}),
		},
	}

	t.Run("Deve correlacionar Git e Terraform corretamente", func(t *testing.T) {
		analysis := Analyse(collected, targets)

		if len(analysis.Components) != 2 {
			t.Fatalf("esperava 2 componentes analisados, obteve %d", len(analysis.Components))
		}

		// Valida se o primeiro componente é 'Add'
		if analysis.Components[0].ChangeType != git.ChangeAdd {
			t.Errorf("esperava Add, obteve %s", analysis.Components[0].ChangeType)
		}

		// Valida o REPLACE (crítico para o seu pipeline)
		if !analysis.Components[1].IsReplacement {
			t.Error("o componente database deveria ter IsReplacement = true")
		}
	})
}

// Helper para criar um JSON de plano do Terraform simulando ações específicas
func mockTFPlanJSON(address string, actions tfjson.Actions) []byte {
	plan := tfjson.Plan{
		FormatVersion: "1.2",
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address: address,
				Change: &tfjson.Change{
					Actions: actions,
				},
			},
		},
	}
	b, _ := json.Marshal(plan)
	return b
}
