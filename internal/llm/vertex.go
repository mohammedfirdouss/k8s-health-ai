package llm

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// VertexProvider calls Gemini on Vertex AI via google.golang.org/genai.
// Set GOOGLE_GENAI_USE_VERTEXAI=true, GOOGLE_CLOUD_PROJECT, and GOOGLE_CLOUD_LOCATION before starting the process.
type VertexProvider struct {
	Client *genai.Client
	Model  string
}

// NewVertexProvider creates a client; Vertex is selected via environment (see genai docs).
func NewVertexProvider(ctx context.Context) (*VertexProvider, error) {
	model := os.Getenv("VERTEX_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{})
	if err != nil {
		return nil, err
	}
	return &VertexProvider{Client: client, Model: model}, nil
}

// Diagnose implements Provider.
func (v *VertexProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	user := BuildUserPrompt(fc)
	parts := genai.Text(user)
	if len(parts) == 0 {
		return Diagnosis{}, fmt.Errorf("empty user content")
	}
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: SystemPrompt()}}},
		Temperature:       genai.Ptr[float32](0.2),
		MaxOutputTokens:   1024,
	}
	resp, err := v.Client.Models.GenerateContent(ctx, v.Model, parts, cfg)
	if err != nil {
		return Diagnosis{}, err
	}
	text := resp.Text()
	d, err := ParseDiagnosisJSON(text)
	if err != nil {
		return Diagnosis{}, err
	}
	if d.Model == "" {
		d.Model = v.Model
	}
	return d, nil
}
