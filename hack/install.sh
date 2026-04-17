#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
kubectl apply -f "${ROOT}/config/crd/clusterdiagnoses.yaml"
kubectl apply -f "${ROOT}/config/manager/namespace.yaml"
kubectl apply -f "${ROOT}/config/rbac/service_account.yaml"
kubectl apply -f "${ROOT}/config/rbac/cluster_role.yaml"
kubectl apply -f "${ROOT}/config/rbac/cluster_role_binding.yaml"
