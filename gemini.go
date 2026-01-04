package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// GeminiClient é o cliente para interagir com a API do Gemini
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient cria uma nova instância do cliente Gemini
// A API key pode ser fornecida via variável de ambiente GEMINI_API_KEY
// ou passada diretamente como parâmetro
func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	ctx := context.Background()

	// Se apiKey não for fornecida, tentar pegar da variável de ambiente
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("API key não fornecida e variável GEMINI_API_KEY não encontrada")
		}
	}

	// Criar configuração do cliente com API key
	config := &genai.ClientConfig{
		APIKey: apiKey,
	}

	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente Gemini: %w", err)
	}

	return &GeminiClient{
		client: client,
		model:  "gemini-2.5-flash", // Modelo padrão
	}, nil
}

// SetModel define o modelo a ser usado
func (g *GeminiClient) SetModel(model string) {
	g.model = model
}

// GetModel retorna o modelo atual
func (g *GeminiClient) GetModel() string {
	return g.model
}

// GenerateContent gera conteúdo de texto usando o Gemini
func (g *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	// Criar conteúdo com o prompt
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}

	// Gerar conteúdo
	response, err := g.client.Models.GenerateContent(ctx, g.model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar conteúdo: %w", err)
	}

	// Extrair texto da resposta
	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		return response.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("resposta vazia do Gemini")
}

// GenerateContentWithHistory gera conteúdo com histórico de conversa
func (g *GeminiClient) GenerateContentWithHistory(ctx context.Context, prompt string, history []*genai.Content) (string, error) {
	// Adicionar o novo prompt ao histórico
	newContent := &genai.Content{
		Parts: []*genai.Part{
			{Text: prompt},
		},
	}

	contents := append(history, newContent)

	// Gerar conteúdo
	response, err := g.client.Models.GenerateContent(ctx, g.model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar conteúdo: %w", err)
	}

	// Extrair texto da resposta
	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		return response.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("resposta vazia do Gemini")
}

// ListAvailableModels retorna uma lista de modelos disponíveis
func (g *GeminiClient) ListAvailableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.5-flash-8b",
	}
}
