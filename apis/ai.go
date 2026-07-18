package apis

import (
	"context"
	"errors"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient interface for mocking
type OpenAIClient interface {
	ParaphraseDescription(text string) (string, error)
	GenerateTitle() (string, error)
}

// OpenAI struct
type OpenAI struct {
	client *openai.Client
}

// NewOpenAI creates a new OpenAIClient
func NewOpenAI(apiKey string) *OpenAI {
	client := openai.NewClientWithBaseURL(apiKey, "https://gateway.arkane.my.id/v1")
	return &OpenAI{client: client}
}

// ParaphraseDescription text using OpenAI
func (o *OpenAI) ParaphraseDescription(text string) (string, error) {
	resp, err := o.client.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a local Javanese/Indonesian marketing copywriter for Facebook Marketplace. Rewrite the given text to make it unique and engaging in casual Indonesian and Javanese slang. Ensure important details (Free delivery/install, trade-in options, location Bantul/Yogyakarta, prices 400k-600k, WA 081354007400) are preserved. Format nicely with emojis.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: text,
			},
		},
	})
	if err != nil {
		return "", errors.New("Failed to get paraphrase: " + err.Error())
	}
	return resp.Choices[0].Message.Content, nil
}

// GenerateTitle generates a unique battery sale title using OpenAI
func (o *OpenAI) GenerateTitle() (string, error) {
	resp, err := o.client.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Generate a short, catchy, unique title in Indonesian for a Facebook Marketplace listing selling vehicle batteries (aki mobil/motor). Keep it under 100 characters. Example: 'Aki Mobil Baru Bergaransi antar pasang gratis'. Do not use quotes in the output.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Generate one title.",
			},
		},
	})
	if err != nil {
		return "", errors.New("Failed to generate title: " + err.Error())
	}
	return resp.Choices[0].Message.Content, nil
}
