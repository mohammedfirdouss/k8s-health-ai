package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const azureOpenAIAPIVersion = "2024-02-15-preview"

// AzureOpenAIProvider calls Azure OpenAI chat completions.
// Env: AZURE_OPENAI_ENDPOINT (base URL without trailing slash), AZURE_OPENAI_API_KEY, AZURE_OPENAI_DEPLOYMENT.
type AzureOpenAIProvider struct {
	Endpoint   string
	APIKey     string
	Deployment string
	Client     *http.Client
}

// NewAzureOpenAIProvider loads configuration from the environment.
func NewAzureOpenAIProvider() (*AzureOpenAIProvider, error) {
	endpoint := strings.TrimSuffix(strings.TrimSpace(os.Getenv("AZURE_OPENAI_ENDPOINT")), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT is required")
	}
	key := strings.TrimSpace(os.Getenv("AZURE_OPENAI_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_API_KEY is required")
	}
	deployment := strings.TrimSpace(os.Getenv("AZURE_OPENAI_DEPLOYMENT"))
	if deployment == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_DEPLOYMENT is required")
	}
	return &AzureOpenAIProvider{
		Endpoint:   endpoint,
		APIKey:     key,
		Deployment: deployment,
		Client:     http.DefaultClient,
	}, nil
}

type azureChatRequest struct {
	Messages  []azureChatMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

type azureChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type azureChatResponse struct {
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
func (a *AzureOpenAIProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	u, err := url.Parse(a.Endpoint + "/openai/deployments/" + url.PathEscape(a.Deployment) + "/chat/completions")
	if err != nil {
		return Diagnosis{}, err
	}
	q := u.Query()
	q.Set("api-version", azureOpenAIAPIVersion)
	u.RawQuery = q.Encode()

	body, err := json.Marshal(azureChatRequest{
		Messages: []azureChatMessage{
			{Role: "system", Content: SystemPrompt()},
			{Role: "user", Content: BuildUserPrompt(fc)},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return Diagnosis{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return Diagnosis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.APIKey)

	resp, err := a.client().Do(req)
	if err != nil {
		return Diagnosis{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Diagnosis{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Diagnosis{}, fmt.Errorf("azure openai chat completions: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var out azureChatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Diagnosis{}, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return Diagnosis{}, fmt.Errorf("azure openai api error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return Diagnosis{}, fmt.Errorf("azure openai: empty choices")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	d, err := ParseDiagnosisJSON(text)
	if err != nil {
		return Diagnosis{}, err
	}
	if d.Model == "" {
		d.Model = a.Deployment
	}
	return d, nil
}

func (a *AzureOpenAIProvider) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return http.DefaultClient
}
