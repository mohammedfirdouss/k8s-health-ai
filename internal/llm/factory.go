package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// NewFromEnv returns the LLM provider selected by LLM_PROVIDER (mock|bedrock|vertex).
func NewFromEnv(ctx context.Context) (Provider, error) {
	p := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	if p == "" {
		p = "mock"
	}
	switch p {
	case "mock":
		return MockProvider{Model: "mock"}, nil
	case "bedrock":
		return NewBedrockProvider(ctx)
	case "vertex":
		return NewVertexProvider(ctx)
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want mock, bedrock, vertex)", p)
	}
}
