package config

import (
	"fmt"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"strings"
	"sync"
	"time"
)

var (
	// instance guarda a única cópia da configuração
	instance *Config
	// once garante que a carga só aconteça uma vez
	once sync.Once
)

type Config struct {
	BaseBranch            string        `mapstructure:"base_branch"`
	OutputDir             string        `mapstructure:"output_dir"`
	ControllerUrl         string        `mapstructure:"controller_url"`
	PipelineActionVar     string        `mapstructure:"pipeline_action_var"`
	Timeout               time.Duration `mapstructure:"timeout"`
	TerragruntBin         string        `mapstructure:"terragrunt_bin"`
	TerraformBin          string        `mapstructure:"terraform_bin"`
	TfPluginCacheDir      string        `mapstructure:"tf_plugin_cache_dir"`
	ComplianceRegistryURI  string        `mapstructure:"compliance_registry_uri"`
	ModuleRegistryRoot     string        `mapstructure:"module_registry_root"`

	// Feature Flags
	Features FeatureFlags `mapstructure:"features"`
}
type FeatureFlags struct {
	SecurityScan    bool   `mapstructure:"security_scan"`
	SecurityScanner string `mapstructure:"security_scanner"` // trivy, snyk, checkov
	TrivyBin        string `mapstructure:"trivy_bin"`
	SnykBin         string `mapstructure:"snyk_bin"`
	SnykToken       string `mapstructure:"snyk_token"`
	CheckovBin      string `mapstructure:"checkov_bin"`
	BlockOnCritical bool   `mapstructure:"block_on_critical"`
	BlockOnHigh     bool   `mapstructure:"block_on_high"`

	PlanRegistration bool `mapstructure:"plan_registration"` // Registrar plans no backend
	PRComments       bool `mapstructure:"pr_comments"`       // Comentar em PRs
}

// IsEnabled verifica se uma feature está habilitada
func (f *FeatureFlags) IsEnabled(featureName string) bool {
	switch featureName {
	case "security_scan":
		return f.SecurityScan
	case "plan_registration":
		return f.PlanRegistration
	case "pr_comments":
		return f.PRComments
	default:
		return false
	}
}

// Get retorna a instância do Config.
// Se for a primeira chamada, ele carrega. Se falhar, ele PANICA (Fail Fast).
// Aplicações não devem rodar se a configuração estiver quebrada.
func Get() *Config {
	once.Do(func() {
		cfg, err := load()
		if err != nil {
			// Em CLIs e Daemons, se a config falha, o app deve morrer imediatamente.
			panic(fmt.Sprintf("❌ Falha fatal ao carregar configurações: %v", err))
		}
		instance = cfg
	})
	return instance
}

// Reload força o recarregamento (útil para testes ou signals como SIGHUP)
func Reload() error {
	cfg, err := load()
	if err != nil {
		return err
	}
	instance = cfg
	return nil
}

func load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("base_branch", "master")
	viper.SetDefault("output_dir", "./dist")
	viper.SetDefault("controller_url", "")
	viper.SetDefault("pipeline_action_var", "PIPELINE_ACTION")
	viper.SetDefault("timeout", "60m")
	viper.SetDefault("tf_plugin_cache_dir", "")
	viper.BindEnv("tf_plugin_cache_dir", "TF_PLUGIN_CACHE_DIR")
	viper.SetDefault("terragrunt_bin", "terragrunt")
	viper.SetDefault("terraform_bin", "terraform")
	viper.SetDefault("compliance_registry_uri", "")
	viper.BindEnv("compliance_registry_uri", "COMPLIANCE_REGISTRY_URI")
	viper.SetDefault("module_registry_root", "")
	viper.BindEnv("module_registry_root", "MODULE_REGISTRY_ROOT")

	// ✅ Defaults - Feature Flags (aninhados com "features.")
	viper.SetDefault("features.security_scan", false)
	viper.SetDefault("features.security_scanner", "trivy")
	viper.SetDefault("features.trivy_bin", "trivy")
	viper.SetDefault("features.snyk_bin", "snyk")
	viper.SetDefault("features.snyk_token", "")
	viper.SetDefault("features.checkov_bin", "checkov")
	viper.SetDefault("features.block_on_critical", true)
	viper.SetDefault("features.block_on_high", false)
	viper.SetDefault("features.plan_registration", true)
	viper.SetDefault("features.pr_comments", false)

	// ✅ Defaults - Feature Flags (aninhados com "features.")
	viper.SetDefault("features.security_scan", false)
	viper.SetDefault("features.security_scanner", "trivy")
	viper.SetDefault("features.trivy_bin", "trivy")
	viper.SetDefault("features.snyk_bin", "snyk")
	viper.SetDefault("features.snyk_token", "")
	viper.SetDefault("features.checkov_bin", "checkov")
	viper.SetDefault("features.block_on_critical", true)
	viper.SetDefault("features.block_on_high", false)
	viper.SetDefault("features.plan_registration", true)
	viper.SetDefault("features.pr_comments", false)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	logger := log.New(log.Setup())
	logger.Info().
		Interface("config", viper.AllSettings()).
		Msg("Configurações carregadas")
	return &cfg, nil
}
