// pkg/command/command.go
package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ServerPlace/iac-runner/pkg/log"
)

// RunOptions define parâmetros extras para a execução
type RunOptions struct {
	Dir         string        // Onde o comando vai rodar (Working Directory)
	Env         []string      // Variáveis de ambiente extras
	LiveOutput  bool          // Se true, mostra output em tempo real
	ErrorOutput *bytes.Buffer // Se fornecido, captura stderr aqui
}

// Run executa um comando genérico e retorna stdout ou erro
func Run(ctx context.Context, binPath string, args []string, opts RunOptions) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	logger := log.FromContext(ctx)
	logger.Debug().Msgf("Running command: %s %s", binPath, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, binPath, args...)

	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	if len(opts.Env) > 0 {
		cmd.Env = append(cmd.Environ(), opts.Env...)
	}

	// Buffers para captura
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if opts.LiveOutput {
		// ✅ stdout: tela (os.Stdout) + captura
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)

		// ✅ stderr: tela (os.Stderr) + captura interna + captura externa (opcional)
		stderrWriters := []io.Writer{os.Stderr, &stderrBuf}
		if opts.ErrorOutput != nil {
			stderrWriters = append(stderrWriters, opts.ErrorOutput)
		}
		cmd.Stderr = io.MultiWriter(stderrWriters...)

	} else {
		// Sem live output: apenas capturar
		cmd.Stdout = &stdoutBuf

		if opts.ErrorOutput != nil {
			cmd.Stderr = io.MultiWriter(&stderrBuf, opts.ErrorOutput)
		} else {
			cmd.Stderr = &stderrBuf
		}
	}

	err := cmd.Run()

	// ✅ Retornar apenas stdout (limpo)
	finalOutput := strings.TrimSpace(stdoutBuf.String())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout executing %s in %s", binPath, opts.Dir)
		}

		// ✅ stderr disponível para mensagens de erro
		stderrOutput := strings.TrimSpace(stderrBuf.String())
		if stderrOutput != "" {
			return finalOutput, fmt.Errorf("command failed: %w (stderr: %s)", err, stderrOutput)
		}

		return finalOutput, fmt.Errorf("command failed: %w", err)
	}

	return finalOutput, nil
}
