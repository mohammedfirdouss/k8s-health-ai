# Configuration

All settings are environment variables unless noted.

## LLM provider

| Variable | Values / meaning |
|----------|------------------|
| `LLM_PROVIDER` | `mock` (default), `bedrock`, `vertex`, `openai`, `azure-openai`, `ollama` |
| `LLM_RPM` | Max LLM calls per minute (default `120`). Set `0` or negative to disable rate limiting. |

### Amazon Bedrock

| Variable | Meaning |
|----------|---------|
| `AWS_REGION` | Region (default `us-east-1` if unset in code paths) |
| `BEDROCK_MODEL_ID` | e.g. `anthropic.claude-3-haiku-20240307-v1:0` |

Use standard AWS credential chain (`AWS_PROFILE`, instance role, etc.).

### Google Vertex (Gemini)

| Variable | Meaning |
|----------|---------|
| `GOOGLE_GENAI_USE_VERTEXAI` | Set to `true` for Vertex |
| `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION` | Project and region |
| `VERTEX_MODEL` | e.g. `gemini-2.0-flash` |

### OpenAI

| Variable | Meaning |
|----------|---------|
| `OPENAI_API_KEY` | Required |
| `OPENAI_BASE_URL` | Default `https://api.openai.com/v1` |
| `OPENAI_MODEL` | Default `gpt-4o-mini` |

### Azure OpenAI

| Variable | Meaning |
|----------|---------|
| `AZURE_OPENAI_ENDPOINT` | Resource endpoint URL (no trailing slash) |
| `AZURE_OPENAI_API_KEY` | API key |
| `AZURE_OPENAI_DEPLOYMENT` | Deployment name |

### Ollama (local)

| Variable | Meaning |
|----------|---------|
| `OLLAMA_HOST` | Default `http://127.0.0.1:11434` |
| `OLLAMA_MODEL` | Default `llama3.2` |

## Operator runtime

| Flag / env | Meaning |
|------------|---------|
| `--metrics-bind-address` | Prometheus metrics (default `:8080`) |
| `--health-probe-bind-address` | Health checks (default `:8081`) |

Custom metric: `diagnoses_total{failure_type,phase}`.

Scaling, HPA caveats, and dev hooks: [operations.md](operations.md).

## Secrets for LLM credentials

API keys should be stored in a Kubernetes Secret rather than hardcoded in the deployment.

**Option 1: Create secret imperatively**

```bash
kubectl -n k8s-health-ai-system create secret generic llm-credentials \
  --from-literal=OPENAI_API_KEY=sk-... \
  --from-literal=AZURE_OPENAI_API_KEY=... \
  --from-literal=AZURE_OPENAI_ENDPOINT=https://... \
  --from-literal=AZURE_OPENAI_DEPLOYMENT=...
```

**Option 2: Apply the sample manifest**

Edit `config/manager/secret-llm.yaml` with your values and apply:

```bash
kubectl apply -f config/manager/secret-llm.yaml
```

> **Warning**: Never commit real API keys to version control. The sample manifest contains empty placeholders only.

The deployment references this secret via `envFrom` with `optional: true`, so the manager will start without it (defaulting to the `mock` provider).

## CLI (`diagctl`)

Uses kubeconfig or in-cluster config like `kubectl`. See `diagctl explain` for a short reference.
