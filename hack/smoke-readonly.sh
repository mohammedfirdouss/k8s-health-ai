#!/usr/bin/env bash
# Verifies the ClusterDiagnosis CRD API is visible (requires kubectl + CRD applied).
set -euo pipefail
if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found" >&2
  exit 1
fi
if ! kubectl api-resources --api-group=health.k8sai.io 2>/dev/null | grep -q clusterdiagnoses; then
  echo "clusterdiagnoses.health.k8sai.io not registered — apply config/crd/clusterdiagnoses.yaml" >&2
  exit 1
fi
echo "OK: health.k8sai.io clusterdiagnoses is registered"
