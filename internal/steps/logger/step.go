package logger

import (
	"context"
	"github.com/ServerPlace/iac-runner/internal/core"
	"github.com/ServerPlace/iac-runner/internal/engine"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"github.com/rs/zerolog"
)

// ====================================================================
// 1. Definição das Opções
// ====================================================================

// options segura a configuração interna do Step
type options struct {
	customLogger *zerolog.Logger
}

// Option define a assinatura da função modificadora
type Option func(*options)

// WithLoggerInstance permite passar um logger já instanciado.
// Útil para testes ou quando o logger já foi criado no main.
func WithLoggerInstance(l zerolog.Logger) Option {
	return func(o *options) {
		o.customLogger = &l
	}
}

// StepInit configura o Zerolog baseado em env vars e o injeta no contexto.
func StepInit(opts ...Option) engine.Step {
	// 1. Processa as opções (Pre-Flight)
	// Inicializa com zero values
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(next engine.Executable) engine.Executable {
		return func(ctx context.Context, state *core.ExecutionState) error {
			var logger zerolog.Logger

			// 2. Decisão da Origem
			if cfg.customLogger != nil {
				// A. Usa a instância fornecida via Option
				logger = *cfg.customLogger
			} else {
				// B. Comportamento Padrão: Cria do zero baseado em Env Vars
				lvl := log.Setup()
				logger = log.New(lvl)
			}

			logger.Info().Msg("📝 Logger configurado no contexto")
			// 4. Injeção no Contexto
			ctxWithLog := log.WithLogger(ctx, logger)

			// 5. Próximo passo
			return next(ctxWithLog, state)
		}
	}
}
