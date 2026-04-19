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

// OpenAIProvider calls OpenAI-compatible Chat Completions (e.g. OpenAI API or compatible proxies).
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// NewOpenAIProvider reads OPENAI_API_KEY (required), OPENAI_BASE_URL (default https://api.openai.com/v1),
// OPENAI_MODEL (default gpt-4o-mini).
func NewOpenAIProvider() (*OpenAIProvider, error) {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		BaseURL: base,
		APIKey:  key,
		Model:   model,
		Client:  http.DefaultClient,
	}, nil
}

type openAIChatRequest struct {
	Model     string              `json:"model"`
	Messages  []openAIChatMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Diagnose implements Provider.
func (o *OpenAIProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	body, err := json.Marshal(openAIChatRequest{
		Model: o.Model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: SystemPrompt()},
			{Role: "user", Content: BuildUserPrompt(fc)},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return Diagnosis{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Diagnosis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

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
		return Diagnosis{}, fmt.Errorf("openai chat completions: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var out openAIChatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Diagnosis{}, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return Diagnosis{}, fmt.Errorf("openai api error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return Diagnosis{}, fmt.Errorf("openai: empty choices")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	d, err := ParseDiagnosisJSON(text)
	if err != nil {
		return Diagnosis{}, err
	}
	if d.Model == "" {
		d.Model = o.Model
	}
	return d, nil
}

func (o *OpenAIProvider) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return http.DefaultClient
}
