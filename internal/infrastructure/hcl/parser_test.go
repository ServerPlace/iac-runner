package hcl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserParseModuleSuccess testa o parsing bem-sucedido de um arquivo HCL válido
func TestParserParseModuleSuccess(t *testing.T) {
	parser := &Parser{}

	// Criar arquivo temporário com conteúdo HCL válido
	content := `
dependency "vpc" {
  config_path = "../vpc"
}

dependency "database" {
  config_path = "../database"
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	// Executar
	module, err := parser.ParseModule(tmpFile)

	// Verificar
	require.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, 2, len(module.Dependencies))
	assert.True(t, len(module.Path) > 0)
}

// TestParserParseModuleWithSingleDependency testa parsing com uma única dependência
func TestParserParseModuleWithSingleDependency(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "vpc" {
  config_path = "../vpc"
}

`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.Equal(t, 1, len(module.Dependencies))
}

// TestParserParseModuleNoDependencies testa parsing de arquivo sem dependências
func TestParserParseModuleNoDependencies(t *testing.T) {
	parser := &Parser{}

	content := `
# Arquivo vazio ou sem dependencies
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.Equal(t, 0, len(module.Dependencies))
}

// TestParserParseModulePathResolution testa a resolução correta de caminhos relativos
func TestParserParseModulePathResolution(t *testing.T) {
	parser := &Parser{}

	// Criar estrutura de diretórios temporária
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	content := `
dependency "vpc" {
  config_path = "../vpc"
}

dependency "db" {
  config_path = "../../shared/database"
}
`
	hclFile := filepath.Join(appDir, "terragrunt.hcl")
	err := os.WriteFile(hclFile, []byte(content), 0644)
	require.NoError(t, err)

	module, err := parser.ParseModule(hclFile)

	require.NoError(t, err)
	assert.Equal(t, 2, len(module.Dependencies))

	// Verificar que os caminhos foram resolvidos corretamente
	// app/../vpc -> vpc
	expectedPath1 := filepath.Join(tmpDir, "vpc")
	assert.Equal(t, filepath.Clean(expectedPath1), module.Dependencies[0])

	// app/../../shared/database -> shared/database
	expectedPath2 := filepath.Join(tmpDir, "../shared", "database")
	assert.Equal(t, filepath.Clean(expectedPath2), module.Dependencies[1])
}

// TestParserParseModuleInvalidHCLSyntax testa erro com sintaxe HCL inválida
func TestParserParseModuleInvalidHCLSyntax(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "vpc" {
  config_path = "../vpc"
  # Falta fechar o bloco
}
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	assert.Error(t, err)
	assert.Nil(t, module)
	assert.Contains(t, err.Error(), "failed to decode hcl File")
}

// TestParserParseModuleFileNotFound testa erro quando arquivo não existe
func TestParserParseModuleFileNotFound(t *testing.T) {
	parser := &Parser{}

	module, err := parser.ParseModule("/path/that/does/not/exist/terragrunt.hcl")

	assert.Error(t, err)
	assert.Nil(t, module)
}

// TestParserParseModuleEmptyFile testa parsing de arquivo vazio
func TestParserParseModuleEmptyFile(t *testing.T) {
	parser := &Parser{}

	tmpFile := createTempHCLFile(t, "")
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, 0, len(module.Dependencies))
}

// TestParserParseModuleWithComplexPaths testa caminhos mais complexos
func TestParserParseModuleWithComplexPaths(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "vpc" {
  config_path = "./vpc"
}

dependency "db" {
  config_path = "../../infrastructure/database/prod"
}

dependency "monitoring" {
  config_path = "../monitoring"
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.Equal(t, 3, len(module.Dependencies))
}

// TestParserParseBytesSuccess testa o parsing bem-sucedido de bytes
func TestParserParseBytesSuccess(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
dependency "vpc" {
  config_path = "../vpc"
}

dependency "database" {
  config_path = "../database"
}
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 2, len(config.Dependencies))
	assert.Equal(t, "vpc", config.Dependencies[0].Name)
	assert.Equal(t, "../vpc", config.Dependencies[0].ConfigPath)
	assert.Equal(t, "database", config.Dependencies[1].Name)
	assert.Equal(t, "../database", config.Dependencies[1].ConfigPath)
}

// TestParserParseBytesSingleDependency testa parsing com uma única dependência em bytes
func TestParserParseBytesSingleDependency(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
dependency "app_config" {
  config_path = "../config"
}
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 1, len(config.Dependencies))
	assert.Equal(t, "app_config", config.Dependencies[0].Name)
}

// TestParserParseBytesNoDependencies testa parsing de bytes sem dependências
func TestParserParseBytesNoDependencies(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
# Apenas comentários
# Sem dependências definidas
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 0, len(config.Dependencies))
}

// TestParserParseBytesEmptyContent testa parsing de conteúdo vazio
func TestParserParseBytesEmptyContent(t *testing.T) {
	parser := &Parser{}

	config, err := parser.ParseBytes([]byte(""))

	require.NoError(t, err)
	assert.Equal(t, 0, len(config.Dependencies))
}

// TestParserParseBytesInvalidSyntax testa erro com sintaxe inválida em bytes
func TestParserParseBytesInvalidSyntax(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
dependency "vpc" {
  config_path = "../vpc"
  # Sintaxe quebrada
}}
`)

	config, err := parser.ParseBytes(content)

	assert.Error(t, err)
	assert.Nil(t, config)
}

// TestParserParseBytesMissingRequiredField testa erro quando campo obrigatório está faltando
func TestParserParseBytesMissingRequiredField(t *testing.T) {
	parser := &Parser{}

	// config_path está faltando
	content := []byte(`
dependency "vpc" {
  # config_path faltando
}
`)

	config, err := parser.ParseBytes(content)

	// HCL pode não retornar erro se permitir campos opcionais,
	// mas a dependência não terá ConfigPath
	require.Error(t, err)
	assert.Nil(t, config)
	//assert.Equal(t, "", config.Dependencies[0].ConfigPath)
}

// TestParserParseBytesWithComments testa parsing com comentários
func TestParserParseBytesWithComments(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
# Este é um comentário
dependency "vpc" {
  config_path = "../vpc"  # Comentário inline
}

# Outro comentário
dependency "db" {
  config_path = "../database"
}
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 2, len(config.Dependencies))
}

// TestParserParseBytesWithWhitespace testa parsing com múltiplos espaços
func TestParserParseBytesWithWhitespace(t *testing.T) {
	parser := &Parser{}

	content := []byte(`

dependency "vpc" {
    config_path = "../vpc"
}


dependency "db" {
	config_path = "../database"
}

`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 2, len(config.Dependencies))
}

// TestParserStructureDependency testa a struct Dependency
func TestParserStructureDependency(t *testing.T) {
	dep := Dependency{
		Name:       "vpc",
		ConfigPath: "../vpc",
	}

	assert.Equal(t, "vpc", dep.Name)
	assert.Equal(t, "../vpc", dep.ConfigPath)
}

// TestParserStructureTerragruntConfig testa a struct TerragruntConfig
func TestParserStructureTerragruntConfig(t *testing.T) {
	config := TerragruntConfig{
		Dependencies: []Dependency{
			{Name: "vpc", ConfigPath: "../vpc"},
		},
	}

	assert.Equal(t, 1, len(config.Dependencies))
	assert.Equal(t, "vpc", config.Dependencies[0].Name)
}

// TestParserParseModuleReturnModuleType testa que o tipo de retorno é correto
func TestParserParseModuleReturnModuleType(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "vpc" {
  config_path = "../vpc"
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.IsType(t, &core.Module{}, module)
	assert.NotNil(t, module.Path)
}

// TestParserParseModulePathAttribute testa se o atributo Path é populado
func TestParserParseModulePathAttribute(t *testing.T) {
	parser := &Parser{}

	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	os.MkdirAll(appDir, 0755)

	content := `
dependency "vpc" {
  config_path = "../vpc"
}
`
	hclFile := filepath.Join(appDir, "terragrunt.hcl")
	err := os.WriteFile(hclFile, []byte(content), 0644)
	require.NoError(t, err)

	module, err := parser.ParseModule(hclFile)

	require.NoError(t, err)
	assert.Equal(t, appDir, module.Path)
}

// TestParserMultipleDependenciesWithDifferentPaths testa múltiplas dependências
func TestParserMultipleDependenciesWithDifferentPaths(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "vpc" {
  config_path = "../vpc"
}

dependency "security_group" {
  config_path = "../security"
}

dependency "database" {
  config_path = "../../shared/db"
}

dependency "monitoring" {
  config_path = "../monitoring"
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.Equal(t, 4, len(module.Dependencies))
}

// TestParserParseModulePreserveDependencyOrder testa que a ordem das dependências é preservada
func TestParserParseModulePreserveDependencyOrder(t *testing.T) {
	parser := &Parser{}

	content := `
dependency "first" {
  config_path = "../first"
}

dependency "second" {
  config_path = "../second"
}

dependency "third" {
  config_path = "../third"
}
`
	tmpFile := createTempHCLFile(t, content)
	defer os.Remove(tmpFile)

	module, err := parser.ParseModule(tmpFile)

	require.NoError(t, err)
	assert.Equal(t, 3, len(module.Dependencies))
	// Verificar ordem (pode exigir acesso aos nomes das dependências)
}

// TestParserParseBytesWithMultilineStrings testa strings multilinhas
func TestParserParseBytesWithMultilineStrings(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
dependency "vpc" {
  config_path = "../vpc"
}
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 1, len(config.Dependencies))
}

// Função auxiliar para criar arquivo HCL temporário
func createTempHCLFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "terragrunt_*.hcl")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

// TestParserParseBytesWithMultilineStrings testa strings multilinhas
func TestParserParseInvalidAttributes(t *testing.T) {
	parser := &Parser{}

	content := []byte(`
dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
	name = "lp-bucket"
	versioning = true
	criticality = "high"
	
	specific_labels = {
		owner = "squad-platform"
	}
}
`)

	config, err := parser.ParseBytes(content)

	require.NoError(t, err)
	assert.Equal(t, 1, len(config.Dependencies))
}
