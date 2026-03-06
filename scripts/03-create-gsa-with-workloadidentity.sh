#!/usr/bin/env bash

export KSA_NAME="kubernetes-service-account-name"
export K8S_NAMESPACE="kubernetes-namespace"
export PROJECT_ID="projeto-id"
export GSA_NAME="google-cloud-service-account-name"

# Cria a Google Service Account
gcloud iam service-accounts create $GSA_NAME \
    --display-name="GSA for iac-runner" \
    --project=$PROJECT_ID

# Associa a GSA acima a permissao workloadIdentityUser
gcloud iam service-accounts add-iam-policy-binding \
  ${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[${K8S_NAMESPACE}/${KSA_NAME}]"


