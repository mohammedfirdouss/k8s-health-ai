# k8s-health-ai

AI-assisted Kubernetes pod failure diagnosis: a small **controller-runtime** operator that detects `CrashLoopBackOff`, `OOMKilled`, and `ImagePullBackOff`, gathers pod spec, events, and logs, calls a pluggable LLM (**mock** / **Amazon Bedrock** / **Vertex AI Gemini**), and writes results to the `ClusterDiagnosis` CRD (`kubectl get diagnoses`).

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

| Env | Meaning |
|-----|---------|
| `LLM_PROVIDER` | `mock` (default), `bedrock`, or `vertex` |
| `AWS_REGION` | Bedrock region |
| `BEDROCK_MODEL_ID` | e.g. `anthropic.claude-3-haiku-20240307-v1:0` |
| `GOOGLE_GENAI_USE_VERTEXAI` | `true` for Vertex |
| `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION` | Vertex project/location |
| `VERTEX_MODEL` | e.g. `gemini-2.0-flash` |

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
- `internal/llm` — mock, Bedrock Converse, Vertex `google.golang.org/genai`
- `config/` — CRD, RBAC, samples, deployment
