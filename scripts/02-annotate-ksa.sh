#!/usr/bin/env bash

export KSA_NAME="kubernetes-service-account-name"
export K8S_NAMESPACE="kubernetes-namespace"
export GSA_NAME="google-cloud-service-account-name"
export PROJECT_ID="projeto-id"

kubectl annotate serviceaccount ${KSA_NAME} \
  --namespace ${K8S_NAMESPACE} \
  iam.gke.io/gcp-service-account=${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com
