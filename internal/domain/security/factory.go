// internal/domain/security/factory.go
package security

import (
	"github.com/ServerPlace/iac-runner/internal/config"
	"github.com/rs/zerolog"
)

// BuildScanner cria o scanner apropriado baseado no config
// Retorna nil se feature desabilitada ou scanner inválido
func BuildScanner(logger zerolog.Logger, cfg *config.Config) Scanner {
	if !cfg.Features.SecurityScan {
		logger.Debug().Msg("Security scan feature disabled")
		return nil
	}

	var scanner Scanner

	switch cfg.Features.SecurityScanner {
	case "trivy":
		logger.Info().Msg("Using Trivy scanner")
		scanner = NewTrivyScanner(cfg.Features.TrivyBin)

	case "snyk":
		if cfg.Features.SnykToken == "" {
			logger.Warn().Msg("Snyk selected but no token provided")
			return nil
		}
		logger.Info().Msg("Using Snyk scanner")
		scanner = NewSnykScanner(cfg.Features.SnykBin, cfg.Features.SnykToken)

	case "checkov":
		logger.Info().Msg("Using Checkov scanner")
		scanner = NewCheckovScanner(cfg.Features.CheckovBin)

	case "":
		logger.Debug().Msg("No scanner specified, using Trivy")
		scanner = NewTrivyScanner(cfg.Features.TrivyBin)

	default:
		logger.Warn().Msgf("Unknown scanner '%s', using Trivy", cfg.Features.SecurityScanner)
		scanner = NewTrivyScanner(cfg.Features.TrivyBin)
	}

	return scanner
}

// GetPolicy retorna a política de segurança do config
func GetPolicy(cfg *config.Config) Policy {
	return Policy{
		BlockOnCritical: cfg.Features.BlockOnCritical,
		BlockOnHigh:     cfg.Features.BlockOnHigh,
		MaxIssues:       0,
	}
}
