package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// NewFromEnv returns the LLM provider selected by LLM_PROVIDER
// (mock|bedrock|vertex|openai|azure-openai|ollama), wrapped with optional rate limiting (LLM_RPM).
func NewFromEnv(ctx context.Context) (Provider, error) {
	p := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	if p == "" {
		p = "mock"
	}
	var inner Provider
	var err error
	switch p {
	case "mock":
		inner = MockProvider{Model: "mock"}
	case "bedrock":
		inner, err = NewBedrockProvider(ctx)
	case "vertex":
		inner, err = NewVertexProvider(ctx)
	case "openai":
		inner, err = NewOpenAIProvider()
	case "azure-openai":
		inner, err = NewAzureOpenAIProvider()
	case "ollama":
		inner, err = NewOllamaProvider()
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (valid: mock, bedrock, vertex, openai, azure-openai, ollama)", p)
	}
	if err != nil {
		return nil, err
	}
	return WrapWithLLMRateLimit(inner), nil
}
