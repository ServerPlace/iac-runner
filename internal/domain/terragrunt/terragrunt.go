// internal/domain/terragrunt/terragrunt.go
package terragrunt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ServerPlace/iac-runner/pkg/cigroup"
	"github.com/ServerPlace/iac-runner/pkg/command"
	"github.com/ServerPlace/iac-runner/pkg/log"
)

var ErrNochange = fmt.Errorf("no changes detected")

// Result representa o resultado de uma operação do Terragrunt
type Result struct {
	Output       string // Output textual do terraform
	PlanFilePath string // Path completo do plan JSON
	JSONPlan     []byte // Bytes do JSON (para análise rápida)
}

// Option é uma função que configura o TerragruntClient
type Option func(*TerragruntClient)

// TerragruntClient é o cliente para executar comandos Terragrunt
type TerragruntClient struct {
	tgBin          string
	tgVersion      string
	tfBin          string
	tfVersion      string
	pluginCacheDir string
	liveOutput     bool
	env            []string
	useExtended    bool
	grouper        cigroup.Grouper
}

// New cria um novo cliente Terragrunt
func New(ctx context.Context, terragruntPath, terraformPath string, opts ...Option) (*TerragruntClient, error) {
	tgVersion, err := initTerragrunt(ctx, terragruntPath)
	if err != nil {
		return nil, fmt.Errorf("invalid terragrunt binary %s: %w", terragruntPath, err)
	}

	tfVersion, err := initTerraForm(ctx, terraformPath)
	if err != nil {
		return nil, fmt.Errorf("invalid terraform binary %s: %w", terraformPath, err)
	}

	client := &TerragruntClient{
		tgBin:     terragruntPath,
		tgVersion: tgVersion,
		tfBin:     terraformPath,
		tfVersion: tfVersion,
		env:       []string{"TF_IN_AUTOMATION=true"},
		grouper:   cigroup.Nop(),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// WithPluginCacheDir permite definir um diretório customizado para o cache
func WithPluginCacheDir(dir string) Option {
	return func(t *TerragruntClient) {
		t.pluginCacheDir = dir
		t.env = append(t.env, fmt.Sprintf("TF_PLUGIN_CACHE_DIR=%s", t.pluginCacheDir))
	}
}

// WithCredentials adiciona credenciais como variáveis de ambiente
func WithCredentials(name, val string) Option {
	return func(t *TerragruntClient) {
		t.env = append(t.env, fmt.Sprintf("%s=%s", name, val))
	}
}

// WithExtendedPlan habilita conversão do plan para JSON
func WithExtendedPlan(use bool) Option {
	return func(t *TerragruntClient) {
		t.useExtended = use
	}
}

// WithLiveOutput controla se o output deve ser mostrado em tempo real
func WithLiveOutput(live bool) Option {
	return func(t *TerragruntClient) {
		t.liveOutput = live
	}
}

// WithGrouper configura o Grouper para emitir seções colapsáveis no CI
func WithGrouper(g cigroup.Grouper) Option {
	return func(t *TerragruntClient) {
		t.grouper = g
	}
}

// Init executa terragrunt init
func (t *TerragruntClient) Init(ctx context.Context, stackPath string) error {
	section := fmt.Sprintf("terragrunt init: %s", stackPath)
	t.grouper.Open(section)
	defer t.grouper.Close(section)

	params := []string{
		"init",
		"--non-interactive", // ✅ CORRIGIDO
	}

	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: t.liveOutput,
	}

	_, err := command.Run(ctx, t.tgBin, params, opts)
	if err != nil {
		return fmt.Errorf("terragrunt init failed: %w", err)
	}

	return nil
}

// Plan executa terraform plan via terragrunt
func (t *TerragruntClient) Plan(ctx context.Context, stackPath string) (*Result, error) {
	logger := log.FromContext(ctx)
	planFile := filepath.Join(stackPath, "tfplan")

	section := fmt.Sprintf("terragrunt plan: %s", stackPath)
	t.grouper.Open(section)
	defer t.grouper.Close(section)

	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: t.liveOutput, // ✅ Output em tempo real
	}

	params := []string{
		"plan",
		"--non-interactive",
		"-out=tfplan",
	}

	// ✅ Se useExtended, adicionar --detailed-exitcode para detectar mudanças
	if t.useExtended {
		params = append(params, "--detailed-exitcode")
	}

	output, err := command.Run(ctx, t.tgBin, params, opts)

	// ✅ Lógica de exit code
	var isChanged = false
	if t.useExtended {
		if err == nil {
			// Exit code 0 = sem mudanças
			return &Result{Output: output}, ErrNochange
		}

		// Exit code 2 = mudanças detectadas (não é erro!)
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() == 2 {
				isChanged = true
				err = nil
			}
		}
	} else if err == nil {
		isChanged = true
	}

	if err != nil {
		return nil, err
	}

	result := &Result{
		Output:       output,
		PlanFilePath: planFile,
	}

	// ✅ Se houve mudanças, gerar JSON
	if isChanged && t.useExtended {
		jsonPath, jsonContent, jsonErr := t.ShowJSON(ctx, stackPath, "tfplan")
		if jsonErr != nil {
			logger.Warn().Err(jsonErr).Msg("Failed to generate JSON plan")
		} else {
			result.PlanFilePath = jsonPath
			result.JSONPlan = jsonContent
		}
	}

	return result, nil
}

// PlanDestroy executa terraform plan -destroy
func (t *TerragruntClient) PlanDestroy(ctx context.Context, stackPath string) (*Result, error) {
	logger := log.FromContext(ctx)
	planFile := filepath.Join(stackPath, "tfplan-destroy")

	section := fmt.Sprintf("terragrunt plan -destroy: %s", stackPath)
	t.grouper.Open(section)
	defer t.grouper.Close(section)

	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: t.liveOutput,
	}

	params := []string{
		"plan",
		"--non-interactive",
		"-destroy",
		"-out=tfplan-destroy",
	}

	if t.useExtended {
		params = append(params, "--detailed-exitcode")
	}

	output, err := command.Run(ctx, t.tgBin, params, opts)

	var isChanged = false
	if t.useExtended {
		if err == nil {
			return &Result{Output: output}, ErrNochange
		}

		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() == 2 {
				isChanged = true
				err = nil
			}
		}
	} else if err == nil {
		isChanged = true
	}

	if err != nil {
		return nil, err
	}

	result := &Result{
		Output:       output,
		PlanFilePath: planFile,
	}

	if isChanged && t.useExtended {
		jsonPath, jsonContent, jsonErr := t.ShowJSON(ctx, stackPath, "tfplan-destroy")
		if jsonErr != nil {
			logger.Warn().Err(jsonErr).Msg("Failed to generate JSON plan")
		} else {
			result.PlanFilePath = jsonPath
			result.JSONPlan = jsonContent
		}
	}

	return result, nil
}

// Apply executa terraform apply
func (t *TerragruntClient) Apply(ctx context.Context, stackPath string) (*Result, error) {
	section := fmt.Sprintf("terragrunt apply: %s", stackPath)
	t.grouper.Open(section)
	defer t.grouper.Close(section)

	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: t.liveOutput,
	}

	params := []string{
		"apply",
		"--non-interactive",
		"-auto-approve",
	}

	output, err := command.Run(ctx, t.tgBin, params, opts)
	if err != nil {
		return nil, fmt.Errorf("terragrunt apply failed: %w", err)
	}

	return &Result{
		Output: output,
	}, nil
}

// Destroy executa terraform destroy
func (t *TerragruntClient) Destroy(ctx context.Context, stackPath string) (*Result, error) {
	section := fmt.Sprintf("terragrunt destroy: %s", stackPath)
	t.grouper.Open(section)
	defer t.grouper.Close(section)

	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: t.liveOutput,
	}

	params := []string{
		"destroy",
		"--non-interactive",
		"-auto-approve",
	}

	output, err := command.Run(ctx, t.tgBin, params, opts)
	if err != nil {
		return nil, fmt.Errorf("terragrunt destroy failed: %w", err)
	}

	return &Result{
		Output: output,
	}, nil
}

// convertToJSON converte o plan binário para JSON
func (t *TerragruntClient) convertToJSON(ctx context.Context, stackPath, planName string) (string, []byte, error) {
	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: false, // ✅ JSON não precisa de live output
	}

	params := []string{
		"show",
		"-json",
		planName,
	}

	jsonOutput, err := command.Run(ctx, t.tgBin, params, opts)
	if err != nil {
		return "", nil, fmt.Errorf("terragrunt show -json failed: %w", err)
	}

	jsonBytes := []byte(jsonOutput)

	// Salvar JSON no disco
	jsonPath := filepath.Join(stackPath, planName+".json")
	if err := os.WriteFile(jsonPath, jsonBytes, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write JSON file: %w", err)
	}

	return jsonPath, jsonBytes, nil
}

// ShowJSON converte o plan binário para JSON e retorna path + bytes
func (t *TerragruntClient) ShowJSON(ctx context.Context, stackPath, planName string) (string, []byte, error) {
	opts := command.RunOptions{
		Dir:        stackPath,
		Env:        t.env,
		LiveOutput: false, // JSON não precisa de live output
	}

	params := []string{
		"show",
		"-json",
		planName,
	}

	jsonOutput, err := command.Run(ctx, t.tgBin, params, opts)
	if err != nil {
		return "", nil, fmt.Errorf("terragrunt show -json failed: %w", err)
	}

	jsonBytes := []byte(jsonOutput)

	// ✅ Nome limpo: plan.tfplan.json ou plan-destroy.tfplan.json
	baseName := "plan"
	if strings.Contains(planName, "destroy") {
		baseName = "plan-destroy"
	}

	jsonPath := filepath.Join(stackPath, baseName+".tfplan.json")

	if err := os.WriteFile(jsonPath, jsonBytes, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write JSON file: %w", err)
	}

	return jsonPath, jsonBytes, nil
}
