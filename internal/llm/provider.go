package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// FailureContext is sent to the LLM.
type FailureContext struct {
	Namespace   string
	Pod         string
	Container   string
	FailureType string
	PodSpecYAML string
	Events      []string
	LogsTail    string
}

// Diagnosis is the structured JSON output from the model.
type Diagnosis struct {
	RootCause      string `json:"root_cause"`
	Severity       string `json:"severity"`
	RecommendedFix string `json:"recommended_fix"`
	Model          string `json:"model"`
}

// Provider calls an LLM backend.
type Provider interface {
	Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error)
}

// BuildUserPrompt renders the user message for diagnosis.
func BuildUserPrompt(fc FailureContext) string {
	var b strings.Builder
	b.WriteString("Analyze this Kubernetes pod failure.\n\n")
	b.WriteString("namespace: ")
	b.WriteString(fc.Namespace)
	b.WriteString("\npod: ")
	b.WriteString(fc.Pod)
	b.WriteString("\ncontainer: ")
	b.WriteString(fc.Container)
	b.WriteString("\nfailure_type: ")
	b.WriteString(fc.FailureType)
	b.WriteString("\n\n--- pod.spec (YAML) ---\n")
	b.WriteString(fc.PodSpecYAML)
	b.WriteString("\n\n--- recent events ---\n")
	for _, e := range fc.Events {
		b.WriteString(e)
		b.WriteByte('\n')
	}
	b.WriteString("\n--- container logs (tail) ---\n")
	b.WriteString(fc.LogsTail)
	b.WriteString("\n")
	return b.String()
}

const systemPrompt = `You are a Kubernetes SRE assistant. Reply with a single JSON object only, no markdown fences, with keys:
root_cause (string, concise hypothesis),
severity (one of: low, medium, high, critical),
recommended_fix (string, concrete kubectl or manifest changes),
model (string, echo the model id you are simulating or "unknown").

If information is insufficient, still return valid JSON with best-effort fields.`

// ParseDiagnosisJSON extracts Diagnosis from model output; retries once if wrapped in prose.
func ParseDiagnosisJSON(raw string) (Diagnosis, error) {
	raw = strings.TrimSpace(raw)
	d, err := tryParseJSON(raw)
	if err == nil {
		return normalize(d), nil
	}
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			d2, err2 := tryParseJSON(raw[i : j+1])
			if err2 == nil {
				return normalize(d2), nil
			}
		}
	}
	return Diagnosis{}, fmt.Errorf("parse diagnosis: %w", err)
}

func tryParseJSON(s string) (Diagnosis, error) {
	var d Diagnosis
	err := json.Unmarshal([]byte(s), &d)
	return d, err
}

func normalize(d Diagnosis) Diagnosis {
	d.Severity = strings.ToLower(strings.TrimSpace(d.Severity))
	switch d.Severity {
	case "low", "medium", "high", "critical":
	default:
		d.Severity = "medium"
	}
	return d
}

// SystemPrompt returns the fixed system instruction.
func SystemPrompt() string { return systemPrompt }
