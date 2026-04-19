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

## CLI (`diagctl`)

Uses kubeconfig or in-cluster config like `kubectl`. See `diagctl explain` for a short reference.
