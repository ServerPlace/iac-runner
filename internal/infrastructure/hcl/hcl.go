package hcl

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"net/url"
	"strings"
)

type hcl2Parser struct {
	*hclparse.Parser
}
type Analisys struct {
	File        *hcl.File
	Diagnostics hcl.Diagnostics
}

func NewAnalisysFromFile(filename string) *Analisys {
	file, diag := hclparse.NewParser().ParseHCLFile(filename)
	return &Analisys{
		file,
		diag,
	}
}
func NewAnalisysFromBytes(src []byte, filename string) *Analisys {
	file, diag := hclparse.NewParser().ParseHCL(src, filename)
	return &Analisys{
		file,
		diag,
	}
}

//func (p *hcl2Parser) ParseHCLFile(filename string) *Analisys {
//	file, diag := p.parser.ParseHCLFile(filename)
//	return &Analisys{
//		file,
//		diag,
//	}
//}
//
//func (p *hcl2Parser) ParseHCL(src []byte, filename string) *Analisys {
//	file, diag := p.parser.ParseHCL(src, filename)
//	return &Analisys{
//		file,
//		diag,
//	}
//}
//func NewHCL2Parser() *hcl2Parser {
//	return &hcl2Parser{parser: hclparse.NewParser()}
//}

func (a *Analisys) ParseInputs() (map[string]hclsyntax.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	body := a.File.Body.(*hclsyntax.Body)
	out := make(map[string]hclsyntax.Expression)

	attr, ok := body.Attributes["inputs"]
	if !ok {
		return out, diags
	}

	obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "inputs inválido",
			Detail:   "inputs deve ser um objeto: inputs = { ... }",
			Subject:  &attr.SrcRange,
		})
		return nil, diags
	}

	for _, item := range obj.Items {
		keyVal, d := item.KeyExpr.Value(nil)
		diags = append(diags, d...)
		if d.HasErrors() {
			continue
		}

		out[keyVal.AsString()] = item.ValueExpr
	}
	return out, diags
}

type SourceRef struct {
	// Base é a URL do repositório (parte antes do "//" separador de módulo).
	// Ex: "https://dev.azure.com/org/proj/_git/blueprints"
	Base string

	// Path é o subdiretório do módulo (a parte após o "//"), sem "/" inicial.
	// Ex: "6-storage/bucket"
	Path string

	// Revision é o valor do ?ref=..., se existir (branch, tag ou sha).
	Revision string

	// Raw é o source original (útil para logs / debug).
	Raw string
}

// ParseTerraformSource encontra e parseia: terraform { source = "..." }
// Retorna (nil, diags) se não existir terraform/source.
// Retorna erro se o source existir mas não for string literal, ou não puder ser parseado.
func (a *Analisys) ParseTerraformSource() (*SourceRef, hcl.Diagnostics) {
	var diags hcl.Diagnostics //= make(hcl.Diagnostics, 0)

	body, ok := a.File.Body.(*hclsyntax.Body)
	if !ok {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "HCL body inesperado",
			Detail:   fmt.Sprintf("body não é *hclsyntax.Body: %T", a.File.Body),
		})
		return nil, diags
	}
	// Procurar bloco terraform { ... }
	var terraformBlock *hclsyntax.Block
	for _, b := range body.Blocks {
		if b.Type == "terraform" {
			terraformBlock = b
			break
		}
	}
	if terraformBlock == nil {
		//diags = append(diags, &hcl.Diagnostic{
		//	Severity: hcl.DiagError,
		//	Summary:  "no terraform block found",
		//	Detail:   fmt.Sprintf("No Terraform Block in File"),
		//})
		// Sem terraform block = sem source (não é erro, depende da sua policy)
		return nil, diags
	}
	// Pegar atributo source dentro do bloco terraform
	attr, ok := terraformBlock.Body.Attributes["source"]
	if !ok {
		// Sem source (não é erro, depende da sua policy)
		return nil, diags
	}
	// Ser parcimonioso: exigir string literal (sem dependências/funções)
	val, vdiags := attr.Expr.Value(nil) // nil context => só passa se for estático
	if vdiags.HasErrors() {
		// Aqui você continua parcimonioso: não aceita referências/funções/interpolação
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "source inválido",
			Detail:   "terraform.source deve ser uma string estática (sem interpolação/funções/refs), ex: source = \"git::https://...//path?ref=v1\"",
			Subject:  &attr.SrcRange,
		}}
	}

	if val.Type().FriendlyName() != "string" {
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "source inválido",
			Detail:   "terraform.source deve resultar em string",
			Subject:  &attr.SrcRange,
		}}
	}

	raw := val.AsString()
	ref, err := parseTerragruntSource(raw)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "source inválido",
			Detail:   err.Error(),
			Subject:  &attr.SrcRange,
		})
		return nil, diags
	}

	ref.Raw = raw
	return ref, diags
}

// parseTerragruntSource extrai Path (após //) e Revision (query param ref=) do source.
func parseTerragruntSource(raw string) (*SourceRef, error) {
	// Terragrunt frequentemente usa prefixos como:
	// - tfr://...
	// - git::https://...
	// - ssh://...
	// url.Parse lida bem com vários, mas "git::" não é um scheme válido,
	// então normalizamos removendo o "git::" (e "hg::", "s3::" etc se quiser).
	s := strings.TrimSpace(raw)

	// Normalização de "git::" (Terraform module source syntax).
	// Se você usa outros "X::", adicione aqui.
	if i := strings.Index(s, "::"); i > 0 {
		prefix := s[:i]
		// Só strip se for um prefix conhecido (evita quebrar URL com "http://").
		switch prefix {
		case "git", "hg", "s3", "gcs":
			s = s[i+2:]
		}
	}

	// Separar base e subdir: base//path?query
	// O “//” é o separador de módulo do Terraform/Terragrunt.
	// Para URLs com protocolo (https://, tfr://), é preciso pular o “://”
	// antes de buscar o separador, para não confundi-lo com o do protocolo.
	basePart := s
	subdirPart := ""
	searchFrom := 0
	if protoEnd := strings.Index(s, "://"); protoEnd >= 0 {
		searchFrom = protoEnd + 3
	}
	if idx := strings.Index(s[searchFrom:], "//"); idx >= 0 {
		splitAt := searchFrom + idx
		basePart = s[:splitAt]
		subdirPart = s[splitAt+2:] // pode conter “?ref=”
	}

	// Para capturar ref corretamente, precisamos parsear a query, que pode estar
	// tanto em basePart quanto em subdirPart (na prática é comum ficar no final).
	// Ex:
	//  basePart = "tfr://github.com/org/repo"
	//  subdirPart = "modules/vpc?ref=v1"
	path := ""
	revision := ""

	// Se subdirPart tem query, parseia.
	if subdirPart != "" {
		if qIdx := strings.Index(subdirPart, "?"); qIdx >= 0 {
			path = subdirPart[:qIdx]
			uq, err := url.ParseQuery(subdirPart[qIdx+1:])
			if err != nil {
				return nil, fmt.Errorf("não consegui parsear query do source: %w", err)
			}
			revision = uq.Get("ref")
		} else {
			path = subdirPart
		}
	}

	// Se não achou ref no subdirPart, tenta achar no basePart (casos raros)
	if revision == "" {
		if qIdx := strings.Index(basePart, "?"); qIdx >= 0 {
			uq, err := url.ParseQuery(basePart[qIdx+1:])
			if err != nil {
				return nil, fmt.Errorf("não consegui parsear query do source: %w", err)
			}
			revision = uq.Get("ref")
			basePart = basePart[:qIdx]
		}
	}

	// Validar minimamente que basePart parece URL/locator (opcional).
	// Se você quiser ser bem permissivo, pode remover isso.
	if _, err := url.Parse(basePart); err != nil {
		// url.Parse raramente falha; mas mantemos segurança.
		return nil, fmt.Errorf("source base inválido: %w", err)
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	return &SourceRef{
		Base:     basePart,
		Path:     path,
		Revision: revision,
		Raw:      raw,
	}, nil
}
