package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ServerPlace/iac-runner/internal/domain/terragrunt"
	"github.com/ServerPlace/iac-runner/pkg/log"
	"os"
	"time"
)

func main() {
	// 1. Definição dos argumentos da CLI
	tgPath := flag.String("tg", "", "Caminho absoluto para o binário do Terragrunt")
	tfPath := flag.String("tf", "", "Caminho absoluto para o binário do Terraform")
	hclPath := flag.String("hcl", "", "Caminho absoluto para o arquivo terragrunt.hcl alvo")
	flag.Parse()

	// 2. Validação básica de entrada
	if *tgPath == "" || *tfPath == "" || *hclPath == "" {
		fmt.Println("Uso incorreto. Exemplo:")
		fmt.Println("go run main.go -tg /usr/local/bin/terragrunt -tf /usr/local/bin/terraform -hcl /caminho/do/projeto/terragrunt.hcl")
		os.Exit(1)
	}

	// 3. Setup do Contexto e Logger
	// Start Logger
	logger := log.New(log.Setup())
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Hour))
	defer cancel()
	ctx = log.WithLogger(ctx, logger)

	fmt.Println("🚀 Iniciando Teste do Cliente Terragrunt")
	fmt.Printf("📁 Terragrunt Bin: %s\n", *tgPath)
	fmt.Printf("📁 Terraform Bin:  %s\n", *tfPath)
	fmt.Printf("📄 Arquivo Alvo:   %s\n", *hclPath)

	opts := []terragrunt.Option{}
	if cacheDir := os.Getenv("TERRAFORM_CACHE"); cacheDir != "" {
		// Garante que o diretório de cache existe (Idempotente)
		logger.Info().Msg("Using cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			logger.Warn().Err(err).Msg("Failed to create plugin cache dir, init might be slower")
		}
		opts = append(opts, terragrunt.WithPluginCacheDir(cacheDir))
	}

	// 4. Instanciação do Cliente (New)
	fmt.Println("\n[1/3] Criando Cliente...")
	client, err := terragrunt.New(ctx, *tgPath, *tfPath, opts...)
	if err != nil {
		logger.Fatal().Err(err).Msg("Falha ao criar instância do Terragrunt")
	}
	fmt.Println("✅ Cliente criado com sucesso!")

	// 5. Execução do Init
	fmt.Println("\n[2/3] Executando Init...")

	err = client.Init(ctx, *hclPath)
	if err != nil {
		fmt.Printf("⚠️ Aviso durante Init (esperado se for placeholder): %v\n", err)
	} else {
		fmt.Println("✅ Init concluído.")
	}

	// 6. Execução do Plan
	fmt.Println("\n[3/3] Executando Plan...")
	// Passando string vazia para credentials por enquanto, pois a assinatura pede
	result, err := client.Plan(ctx, *hclPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Falha crítica no Plan")
	}

	fmt.Println("✅ Plan concluído com sucesso!")
	fmt.Println("--- OUTPUT DO PLAN ---")
	fmt.Println(result.Output)
	fmt.Println("----------------------")
}
