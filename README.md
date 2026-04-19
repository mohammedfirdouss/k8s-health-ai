# k8s-health-ai

AI-assisted Kubernetes pod failure diagnosis: a small **controller-runtime** operator that detects failing pods (including init errors and many pending/scheduling cases), gathers pod spec, events, and logs, calls a pluggable LLM, and writes results to the `ClusterDiagnosis` CRD (`kubectl get diagnoses`). Status can include **resource usage** (requests/limits) and **canned remediations** per failure class. Prometheus metric: `diagnoses_total`.

Related tooling from [awesome-go](https://github.com/avelino/awesome-go): local clusters with [**kind**](https://github.com/kubernetes-sigs/kind), optional image builds with [**ko**](https://github.com/google/ko) instead of the included `Dockerfile`.

## Quick start (local manager + mock LLM)

```bash
make install          # CRD + RBAC (cluster-wide)
make build
export KUBECONFIG=... # default cluster
LLM_PROVIDER=mock ./bin/manager
```

In another terminal:

```bash
kubectl apply -f config/samples/crashloop.yaml
kubectl get diagnoses -n default
kubectl describe clusterdiagnosis -n default
```

## Configuration

Full tables: [docs/configuration.md](docs/configuration.md).

| Env | Meaning |
|-----|---------|
| `LLM_PROVIDER` | `mock` (default), `bedrock`, `vertex`, `openai`, `azure-openai`, `ollama` |
| `LLM_RPM` | LLM calls per minute (default `120`; `0` disables rate limiting) |
| `AWS_REGION`, `BEDROCK_MODEL_ID` | Bedrock |
| `GOOGLE_GENAI_USE_VERTEXAI`, `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`, `VERTEX_MODEL` | Vertex |
| `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL` | OpenAI-compatible API |
| `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_DEPLOYMENT` | Azure OpenAI |
| `OLLAMA_HOST`, `OLLAMA_MODEL` | Local Ollama |

## CLI (`diagctl`)

Non-interactive helper for agents and scripts:

```bash
make build-diagctl
./bin/diagctl list
./bin/diagctl get diag-xxxx -n default
./bin/diagctl explain
```

## Troubleshooting

See [docs/troubleshooting.md](docs/troubleshooting.md) (RBAC, LLM errors, metrics).

## Operations (scaling, hooks, smoke checks)

See [docs/operations.md](docs/operations.md). Quick API check after `make install`:

```bash
make smoke-readonly
```

## Optional manifests

- HorizontalPodAutoscaler: `config/manager/hpa.yaml` (needs metrics-server; see operations doc before enabling multiple replicas).
- Namespace-scoped RBAC sample: `config/rbac/role_namespaced.yaml`, `role_binding_namespaced.yaml`.

## Kind cluster (optional)

```bash
make kind-up          # requires kind CLI
make install
make build && LLM_PROVIDER=mock ./bin/manager
```

## In-cluster deployment

```bash
make deploy           # build image; load into kind if present
```

## Project layout

- `cmd/manager` — operator entrypoint
- `api/v1alpha1` — `ClusterDiagnosis` types
- `internal/controller` — Pod reconciler
- `internal/detect` — failure classification
- `internal/collect` — spec, events, logs
- `internal/llm` — mock, Bedrock, Vertex, OpenAI, Azure OpenAI, Ollama; optional rate limit
- `internal/remediation` — short hints by failure type
- `internal/metrics` — `diagnoses_total`
- `cmd/diagctl` — list/get/delete/explain for CRDs
- `config/` — CRD, RBAC, samples, deployment, optional HPA
