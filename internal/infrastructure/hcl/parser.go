package hcl

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"path/filepath"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/hashicorp/hcl/v2"
)

// Estruturas de Mapeamento do HCL
// Isso diz ao decodificador o que procurar dentro do terragrunt.hcl

type TerragruntConfig struct {
	// Captura blocos: dependency "vpc" { config_path = "..." }
	Dependencies []Dependency `hcl:"dependency,block"`

	// Captura o resto do arquivo para não dar erro de "atributo desconhecido"
	Remaining hcl.Body `hcl:",remain"`
}

type Dependency struct {
	Name            string   `hcl:"name,label"` // O label do bloco (ex: "vpc")
	ConfigPath      string   `hcl:"config_path"`
	ExtraAttributes hcl.Body `hcl:",remain"`
}

// Parser é a struct que será instanciada
type Parser struct{}

// ParseModule lê um arquivo do DISCO e retorna as dependências (Usado no Prepare/Orchestrator)
func (p *Parser) ParseModule(path string) (*core.Module, error) {
	var config TerragruntConfig

	// hclsimple facilita muito a vida: abre arquivo, lê, faz parse e decode na struct
	err := hclsimple.DecodeFile(path, nil, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hcl File %s: %w", path, err)
	}

	mod := &core.Module{Path: filepath.Dir(path)}

	// Converte para o modelo de domínio (Core)
	for _, dep := range config.Dependencies {
		// Resolve o caminho: Se estou em "app/" e config_path é "../vpc",
		// Join("app", "../vpc") -> "vpc"
		fullPath := filepath.Clean(filepath.Join(mod.Path, dep.ConfigPath))
		mod.Dependencies = append(mod.Dependencies, fullPath)
	}

	return mod, nil
}

// ParseBytes lê de MEMÓRIA (slice de bytes) (Usado no Safety Check)
// Serve para validar sintaxe sem precisar salvar no disco.
func (p *Parser) ParseBytes(content []byte) (*TerragruntConfig, error) {
	var config TerragruntConfig

	// Usa um nome de arquivo fictício apenas para mensagens de erro
	err := hclsimple.Decode("terragrunt.hcl", content, nil, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
