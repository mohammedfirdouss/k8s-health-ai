# Troubleshooting

## RBAC: `clusterdiagnoses` forbidden

Ensure `ClusterRole` / `ClusterRoleBinding` from `config/rbac/` are applied and the manager `ServiceAccount` matches the deployment. For namespace-only access, apply the sample `config/rbac/role_namespaced.yaml` and `role_binding_namespaced.yaml` in the target namespace and bind your user or a dedicated service account.

## LLM API errors

- **Throttling / 429**: Lower concurrency or increase `LLM_RPM` cap only if your provider allows higher throughput; otherwise back off at the provider.
- **Auth errors**: Verify API keys and region/project for the selected `LLM_PROVIDER`.
- **Secret not found / empty keys**: Ensure `llm-credentials` secret exists in `k8s-health-ai-system` namespace and contains non-empty values for the provider you're using. See [configuration.md](configuration.md#secrets-for-llm-credentials).
- **Azure OpenAI**: Confirm `AZURE_OPENAI_ENDPOINT` matches the resource (no path suffix) and the deployment name exists.
- **Ollama**: Ensure the daemon is reachable from the manager pod (often requires `hostNetwork` or a sidecar in cluster — not configured by default).

## Diagnosis stuck in `Analyzing`

The reconciler rate-limits repeat LLM calls per diagnosis (annotation `health.k8sai.io/last-llm-call`). Wait for the requeue window or check manager logs for gather/LLM errors.

## No `ClusterDiagnosis` created

The controller only reconciles pods that match a supported failure class (see `diagctl explain`). Confirm the pod shows `CrashLoopBackOff`, `OOMKilled`, `ImagePullBackOff`, init failures, or pending scheduling as implemented in `internal/detect`.

## Metrics not visible

Scrape the manager pod’s metrics port (`8080` by default) or port-forward: `kubectl -n k8s-health-ai-system port-forward deploy/k8s-health-ai-manager 8080:8080` and open `/metrics`.
