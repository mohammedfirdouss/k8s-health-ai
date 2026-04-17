package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// BedrockProvider uses Amazon Bedrock Converse API (Claude).
type BedrockProvider struct {
	Client  *bedrockruntime.Client
	ModelID string
}

// NewBedrockProvider loads default AWS config and creates a runtime client.
func NewBedrockProvider(ctx context.Context) (*BedrockProvider, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	model := os.Getenv("BEDROCK_MODEL_ID")
	if model == "" {
		model = "anthropic.claude-3-haiku-20240307-v1:0"
	}
	return &BedrockProvider{
		Client:  bedrockruntime.NewFromConfig(cfg),
		ModelID: model,
	}, nil
}

// Diagnose implements Provider.
func (b *BedrockProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	user := BuildUserPrompt(fc)
	in := &bedrockruntime.ConverseInput{
		ModelId: aws.String(b.ModelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: user},
				},
			},
		},
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: SystemPrompt()},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(1024),
		},
	}
	out, err := b.Client.Converse(ctx, in)
	if err != nil {
		return Diagnosis{}, err
	}
	text, err := extractConverseText(out)
	if err != nil {
		return Diagnosis{}, err
	}
	d, err := ParseDiagnosisJSON(text)
	if err != nil {
		return Diagnosis{}, err
	}
	if d.Model == "" {
		d.Model = b.ModelID
	}
	return d, nil
}

func extractConverseText(out *bedrockruntime.ConverseOutput) (string, error) {
	if out == nil || out.Output == nil {
		return "", fmt.Errorf("empty converse output")
	}
	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected output type")
	}
	var parts []string
	for _, block := range msg.Value.Content {
		if tb, ok := block.(*types.ContentBlockMemberText); ok {
			parts = append(parts, tb.Value)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no text in message")
	}
	var acc string
	for _, p := range parts {
		acc += p
	}
	return acc, nil
}
