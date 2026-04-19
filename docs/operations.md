# Operations

## Metrics and scaling

### Horizontal Pod Autoscaler (`config/manager/hpa.yaml`)

The sample HPA targets **CPU utilization** on the manager `Deployment`. It only helps when:

- **metrics-server** is installed and serving `Pod` metrics.
- CPU is actually the bottleneck (reconcile loops are often idle waiting on the API server or LLM).

This operator runs **without leader election** (`LeaderElection: false` in `cmd/manager/main.go`). Multiple replicas will each run the same watches and reconcilers. That can cause **duplicate work** (extra API calls, duplicate LLM attempts until annotations rate-limit), not higher throughput for a single pod’s diagnosis.

**Practical guidance:**

- Prefer **one replica** unless you add leader election and/or shard work.
- Treat the HPA manifest as **optional** and cluster-specific; do not assume more pods linearly improve diagnosis latency.

### Prometheus

Scrape the manager metrics listener (default `--metrics-bind-address=:8080`). Counter `diagnoses_total{failure_type,phase}` increments on successful diagnosis (`phase=ready`) and on reconcile failures after LLM/gather errors (`phase=error`).

## Cursor hooks (developers)

Project hooks live under `.cursor/`:

| Hook | Purpose |
|------|---------|
| `afterFileEdit` → `verify-go.sh` | After editing `*.go`, runs a **scoped** `go build` on the touched package when possible (falls back to `go build ./...`), with a **90s** `timeout` when `timeout(1)` exists. |
| `subagentStop` → `subagent-stop.sh` | No-op build (avoids doubling compile work after subagents). |

If hooks misbehave, remove the `matcher` in `hooks.json` temporarily or disable hooks in Cursor settings while debugging.

## Cluster smoke check

With `kubectl` configured and the CRD installed (`make install` or apply `config/crd/clusterdiagnoses.yaml`):

```bash
./hack/smoke-readonly.sh
```

This only checks that the `health.k8sai.io` API is registered; it does not run the manager.

## Audit logging

Kubernetes **API server audit logs** capture all CR create/update/delete events for `clusterdiagnoses.health.k8sai.io`. Configure your cluster's audit policy to log at the `Metadata` or `Request` level for the `health.k8sai.io` group if you need compliance records.

The operator itself does not emit separate audit logs. If you require application-level audit (e.g., "which pod triggered which diagnosis"), you can:

- Query `ClusterDiagnosis` objects (`.spec.targetRef` identifies the pod).
- Add an annotation in the reconciler with the manager pod name / timestamp if traceability is critical.

## NetworkPolicy

For clusters with strict egress controls, apply `config/manager/networkpolicy.yaml`. It allows:

- **Ingress**: Prometheus scrape on port 8080.
- **Egress**: DNS (kube-dns), kube-apiserver (443/6443), LLM provider HTTPS (443), Ollama (11434).

Tighten `ipBlock` CIDRs to your actual API server and LLM provider ranges in production.
