# ===============================
# Go toolchain auto-detect (from go.mod)
# ===============================
# ===============================
# Go toolchain auto-detect (robusto, sem depender de toolchain no go.mod)
# ===============================

GO_MOD_FILE := go.mod

# Extrai "toolchain go1.xx.y" se existir (patch opcional)
GO_TOOLCHAIN_FROM_MOD := $(strip $(shell \
	if [ -f "$(GO_MOD_FILE)" ]; then \
	  awk '/^toolchain[[:space:]]+go[0-9]+\.[0-9]+(\.[0-9]+)?$$/{print $$2; exit}' "$(GO_MOD_FILE)"; \
	fi \
))

# Extrai "go 1.xx" OU "go 1.xx.y"
GO_VERSION_FROM_MOD := $(strip $(shell \
	if [ -f "$(GO_MOD_FILE)" ]; then \
	  awk '/^go[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?$$/{print $$2; exit}' "$(GO_MOD_FILE)"; \
	fi \
))

# Normaliza para major.minor (remove patch se existir)
GO_MAJOR_MINOR := $(strip $(shell \
	if [ -n "$(GO_VERSION_FROM_MOD)" ]; then \
	  echo "$(GO_VERSION_FROM_MOD)" | awk -F. 'NF>=2{print $$1"."$$2}'; \
	fi \
))

# Se houver toolchain no go.mod, usa ela;
# senão, se houver "go 1.xx", monta go<major>.<minor>.0;
# senão, deixa vazio (não exporta GOTOOLCHAIN)
GO_TOOLCHAIN ?= $(strip \
  $(if $(GO_TOOLCHAIN_FROM_MOD),$(GO_TOOLCHAIN_FROM_MOD), \
    $(if $(GO_MAJOR_MINOR),go$(GO_MAJOR_MINOR).0,) \
  ) \
)
GO_TOOLCHAIN := go$(strip $(GO_VERSION_FROM_MOD))

# Só exporta se tiver valor válido (evita "go.0")
ifneq ($(GO_TOOLCHAIN),)
export GOTOOLCHAIN := $(GO_TOOLCHAIN)
endif


# --- Configurações Globais ---
BINARY_DIR := app
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d-%H%M UTC')
PKG := github.com/ServerPlace/iac-runner/pkg/version

# Flags do Linker (Compartilhadas)
LDFLAGS := -s -w \
    -X '$(PKG).Version=$(VERSION)' \
    -X '$(PKG).BuildTime=$(BUILD_TIME)'

# --- Binário 1: CLI Principal ---
CLI_NAME := iac-runner
CLI_SRC  := cmd/iac-cli/main.go

# --- Binário 2: Terragrunt Wrapper ---
TG_NAME := lp-tg
TG_SRC  := cmd/terragrunt/main.go

# --- Alvos (Targets) ---

.PHONY: all
all: build

# Compila TODOS os binários
.PHONY: build
build: go-toolchain build-cli build-tg

# Compila apenas o CLI Principal
.PHONY: build-cli

go-toolchain:
	@echo "toolchain(go.mod) = $(GO_TOOLCHAIN_FROM_MOD)"
	@echo "go(go.mod)        = $(GO_VERSION_FROM_MOD)"
	@echo "GO_TOOLCHAIN      = $(GO_TOOLCHAIN)"
	@echo "GOTOOLCHAIN(env)  = $(GOTOOLCHAIN)"
	@go version

build-cli:
	@echo "🚧 Compilando CLI Principal ($(CLI_NAME))..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(CLI_NAME) $(CLI_SRC)
	@echo "✅ CLI compilado: $(BINARY_DIR)/$(CLI_NAME)"

# Compila apenas o Terragrunt Wrapper
.PHONY: build-tg
build-tg:
	@echo "🚧 Compilando Terragrunt Wrapper ($(TG_NAME))..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(TG_NAME) $(TG_SRC)
	@echo "✅ Wrapper compilado: $(BINARY_DIR)/$(TG_NAME)"

# Executa o CLI Principal (Padrão para run)
.PHONY: run
run: build-cli
	@echo "🚀 Executando $(CLI_NAME)..."
	./$(BINARY_DIR)/$(CLI_NAME)

# Limpa todos os binários
.PHONY: clean
clean:
	@echo "🧹 Removendo binários..."
	rm -f $(BINARY_DIR)/$(CLI_NAME)
	rm -f $(BINARY_DIR)/$(TG_NAME)
	@echo "✨ Limpeza concluída."

.PHONY: help
help:
	@echo "Uso:"
	@echo "  make all        - Compila todos os binários"
	@echo "  make build      - Compila todos os binários (igual all)"
	@echo "  make build-cli  - Compila apenas o iac-runner"
	@echo "  make build-tg   - Compila apenas o lp-tg (terragrunt wrapper)"
	@echo "  make run        - Compila e roda o CLI principal"
	@echo "  make clean      - Remove os executáveis"
