package apis

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

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
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://gateway.arkane.my.id/v1"
	client := openai.NewClientWithConfig(config)
	return &OpenAI{client: client}
}

// ParaphraseDescription text using OpenAI
func (o *OpenAI) ParaphraseDescription(text string) (string, error) {
	resp, err := o.client.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model: "combo-1",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a Indonesian marketing copywriter for Facebook Marketplace. Rewrite the given text to make it unique and engaging in casual Indonesian slang. Ensure important details (Free delivery/install, trade-in options, location Bantul/Yogyakarta, prices 400k-600k, WA 081354007400) are preserved. Format without emoji and styling like ** for bold cause facebook cant read it.",
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
	angles := []string{
		"aki soak bisa tukar tambah",
		"antar pasang gratis area Jogja Bantul",
		"aki baru bergaransi resmi",
		"konsultasi aki mobil motor",
		"harga aki mulai 400 ribuan",
		"aki mobil motor sepeda listrik",
		"stok banyak merk dan tipe",
	}
	tone := []string{"natural", "urgent", "ramah", "singkat", "lokal Jogja", "anti kaku"}
	seed := time.Now().UnixNano()
	angle := angles[rand.Intn(len(angles))]
	style := tone[rand.Intn(len(tone))]
	resp, err := o.client.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model:       "combo-1",
		Temperature: 1.15,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Buat 1 judul Facebook Marketplace bahasa Indonesia untuk jual aki kendaraan. Maksimal 80 karakter. Jangan pakai tanda kutip. Jangan pakai template umum seperti 'Aki Mobil Baru Bergaransi antar pasang gratis'. Jangan selalu mulai dengan 'Aki Mobil'. Variasikan kata pembuka, susunan, dan fokus manfaat. Output hanya judul.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("Fokus: %s. Tone: %s. Seed unik: %d.", angle, style, seed),
			},
		},
	})
	if err != nil {
		return "", errors.New("Failed to generate title: " + err.Error())
	}
	return resp.Choices[0].Message.Content, nil
}
