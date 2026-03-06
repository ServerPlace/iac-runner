#!/usr/bin/env bash
set -euo pipefail

# =========================
# Validação obrigatória
# =========================
: "${AZP_URL:?Missing AZP_URL}"
: "${AZP_TOKEN:?Missing AZP_TOKEN}"
: "${AZP_POOL:?Missing AZP_POOL}"

AZP_AGENT_NAME="${AZP_AGENT_NAME:-$(hostname)}"

echo ">> Configuring Azure Pipelines agent: $AZP_AGENT_NAME"

# =========================
# Cleanup sempre
# =========================
cleanup() {
  echo ">> Cleanup agent"
  ./config.sh remove --unattended \
    --auth pat \
    --token "$AZP_TOKEN" || true
}
trap cleanup EXIT

# =========================
# Configura agente (1 job)
# =========================
./config.sh --unattended \
  --agent "$AZP_AGENT_NAME" \
  --url "$AZP_URL" \
  --auth pat \
  --token "$AZP_TOKEN" \
  --pool "$AZP_POOL" \
  --work _work \
  --replace \
  --acceptTeeEula

# =========================
# Executa exatamente 1 job
# =========================
./run.sh --once

