package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// OllamaProvider calls Ollama's /api/chat endpoint.
type OllamaProvider struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

// NewOllamaProvider reads OLLAMA_HOST (default http://127.0.0.1:11434) and OLLAMA_MODEL (default llama3.2).
func NewOllamaProvider() (*OllamaProvider, error) {
	host := strings.TrimSuffix(strings.TrimSpace(os.Getenv("OLLAMA_HOST")), "/")
	if host == "" {
		host = "http://127.0.0.1:11434"
	}
	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		BaseURL: host,
		Model:   model,
		Client:  http.DefaultClient,
	}, nil
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// Diagnose implements Provider.
func (o *OllamaProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	body, err := json.Marshal(ollamaChatRequest{
		Model: o.Model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: SystemPrompt()},
			{Role: "user", Content: BuildUserPrompt(fc)},
		},
		Stream: false,
	})
	if err != nil {
		return Diagnosis{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Diagnosis{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client().Do(req)
	if err != nil {
		return Diagnosis{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Diagnosis{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Diagnosis{}, fmt.Errorf("ollama /api/chat: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var out ollamaChatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Diagnosis{}, err
	}
	text := strings.TrimSpace(out.Message.Content)
	d, err := ParseDiagnosisJSON(text)
	if err != nil {
		return Diagnosis{}, err
	}
	if d.Model == "" {
		d.Model = o.Model
	}
	return d, nil
}

func (o *OllamaProvider) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return http.DefaultClient
}
