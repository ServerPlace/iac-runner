#!/usr/bin/env bash

# Definição de variáveis de ambiente
export REPO_URL="<YOUR_DOCKER_REPOSITORY_URL>"
export IMAGE_TAG="v1-$(git rev-parse --short HEAD)"

echo ">>> Iniciando deploy da versão: ${IMAGE_TAG}"

# Construção da imagem
docker build -t "${REPO_URL}:${IMAGE_TAG}" \
  --ssh default \
  --build-arg CACHEBUST=$(date +%s) \
  --platform linux/amd64 .

# Verifique se não é necessário algum comando de login de registry neste ponto

# Publicação no registry
docker push "${REPO_URL}:${IMAGE_TAG}"
