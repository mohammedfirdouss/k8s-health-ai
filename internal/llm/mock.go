package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// MockProvider returns deterministic JSON for local development.
type MockProvider struct {
	Model string
}

// Diagnose implements Provider.
func (m MockProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	_ = ctx
	model := m.Model
	if model == "" {
		model = "mock"
	}
	d := Diagnosis{
		RootCause: fmt.Sprintf("Mock analysis for %s on pod %s/%s: failure %s — check container command, resources, and image name.",
			fc.Container, fc.Namespace, fc.Pod, fc.FailureType),
		Severity:       "high",
		RecommendedFix: "Inspect events and logs above; adjust limits or image reference; verify registry credentials for image pull failures.",
		Model:          model,
	}
	if _, err := json.Marshal(d); err != nil {
		return Diagnosis{}, err
	}
	return d, nil
}
