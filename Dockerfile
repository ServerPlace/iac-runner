FROM golang:1.24-alpine AS iac-builder

# 2. Instale o Git (o Alpine não vem com ele por padrão)
RUN apk add --no-cache git openssh-client make

RUN mkdir -p -m 0700 ~/.ssh && \
    ssh-keyscan github.com >> ~/.ssh/known_hosts

# 3. Defina o diretório de trabalho
WORKDIR /build

# 4. Clone o repositório (Substitua pela URL real)
# DICA: Usamos um ARG para "quebrar" o cache do Docker se precisar forçar update
ARG CACHEBUST=1
COPY . .
# 5. Baixe dependências e faça o Build
#@RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
#RUN --mount=type=ssh \
#    go env -w GOPRIVATE='github.com/ServerPlace/*' && \
RUN go mod download
RUN make build


FROM ubuntu:22.04

LABEL org.opencontainers.image.source=http://github.com/ServerPlace/iac-runner
LABEL org.opencontainers.image.description="IaC Runner"
LABEL org.opencontainers.image.licenses="GPL V3"

ENV DEBIAN_FRONTEND=noninteractive

# =========================
# 1. Dependências base
# =========================
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    jq \
    wget \
    unzip \
    gnupg \
    lsb-release \
    python3 \
    python3-pip \
    python3-venv \
    openssh-client \
    npm \
    && rm -rf /var/lib/apt/lists/*

# =========================
# 2. Terraform
# =========================
ARG TERRAFORM_VERSION=1.14.3
RUN curl -fsSL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip \
    -o terraform.zip \
 && unzip terraform.zip \
 && mv terraform /usr/local/bin/terraform \
 && rm terraform.zip

# =========================
# 3. Terragrunt
# =========================
ARG TERRAGRUNT_VERSION=0.98.0
RUN curl -fsSL https://github.com/gruntwork-io/terragrunt/releases/download/v${TERRAGRUNT_VERSION}/terragrunt_linux_amd64 \
    -o /usr/local/bin/terragrunt \
 && chmod +x /usr/local/bin/terragrunt

# =========================
# 4. Google Cloud CLI
# =========================
RUN curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg \
 && echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    > /etc/apt/sources.list.d/google-cloud-sdk.list \
 && apt-get update \
 && apt-get install -y google-cloud-cli \
 && rm -rf /var/lib/apt/lists/*

# ==============================================================================
# 5. INSTALL SECURITY SCANNERS
# ==============================================================================

# Trivy - Security Scanner (Open Source, fast)
ARG TRIVY_VERSION=0.68.1
ARG SNYK_VERSION=1.1302.1
ARG CHECKOV_VERSION=3.2.500

ARG TRIVY_VERSION
RUN wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor -o /usr/share/keyrings/trivy.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | tee /etc/apt/sources.list.d/trivy.list && \
    apt-get update && \
    apt-get install -y trivy && \
    rm -rf /var/lib/apt/lists/* && \
    trivy --version

# ==============================================================================
# INSTALL SNYK
# ==============================================================================
ARG SNYK_VERSION
RUN npm install -g snyk@${SNYK_VERSION} && \
    snyk --version

# ==============================================================================
# INSTALL CHECKOV
# ==============================================================================
ARG CHECKOV_VERSION
# Option 1: Using virtual environment (recommended for Docker)
RUN python3 -m venv /opt/checkov-venv && \
    /opt/checkov-venv/bin/pip install --no-cache-dir checkov==${CHECKOV_VERSION} && \
    ln -s /opt/checkov-venv/bin/checkov /usr/local/bin/checkov && \
    checkov --version

# =========================
# 6. Azure Pipelines Agent
# =========================
ENV AZP_AGENT_VERSION=4.268.0
WORKDIR /azp

RUN curl -fsSL https://download.agent.dev.azure.com/agent/${AZP_AGENT_VERSION}/vsts-agent-linux-x64-${AZP_AGENT_VERSION}.tar.gz \
  | tar zx

RUN apt-get update && apt-get install -y \
    libicu70 \
    libssl3 \
    libkrb5-3 \
    zlib1g \
    liblttng-ust1 \
    libcurl4 \
 && rm -rf /var/lib/apt/lists/*

# Script oficial do agent para garantir dependências restantes
RUN /azp/bin/installdependencies.sh

# =========================
# 6. Usuário não-root (boa prática)
# =========================
RUN useradd -m iac-user
RUN chown -R iac-user:iac-user /azp

# =============================
# 5. LP IAC CLI
# =============================
COPY --from=iac-builder --chown=iac-user:iac-user --chmod=755  /build/app/iac-runner /usr/local/bin/iac-runner
COPY --from=iac-builder --chown=iac-user:iac-user --chmod=755  /build/app/lp-tg /usr/local/bin/lp-tg


# =========================
# 7. Entrypoint
# =========================
COPY --chown=iac-user:iac-user start.sh /azp/start.sh
RUN chmod +x /azp/start.sh
USER iac-user
ENTRYPOINT ["/azp/start.sh"]
