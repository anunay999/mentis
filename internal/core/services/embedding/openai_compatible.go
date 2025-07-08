package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anunay/mentis/internal/config"
)

type OpenAICompatibleProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAICompatibleProvider(cfg config.OpenAICompatibleConfig) (*OpenAICompatibleProvider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required for OpenAI-compatible provider")
	}

	// Ensure base URL ends with /v1 if it doesn't already
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}

	return &OpenAICompatibleProvider{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Reuse the same request/response structures as OpenAI
type CompatibleEmbeddingRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

type CompatibleEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAICompatibleProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

func (p *OpenAICompatibleProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := CompatibleEmbeddingRequest{
		Input:          texts,
		Model:          p.model,
		EncodingFormat: "float",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Add authorization header if API key is provided
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp CompatibleEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	embeddings := make([][]float32, len(embeddingResp.Data))
	for i, data := range embeddingResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

func (p *OpenAICompatibleProvider) GetDimensions() int {
	// This varies by model and provider
	// Common dimensions for different models:
	switch {
	case strings.Contains(p.model, "nomic-embed"):
		return 768
	case strings.Contains(p.model, "all-MiniLM"):
		return 384
	case strings.Contains(p.model, "bge-"):
		return 1024
	case strings.Contains(p.model, "e5-"):
		return 1024
	default:
		return 1536 // Default to OpenAI standard
	}
}

func (p *OpenAICompatibleProvider) GetModelName() string {
	return p.model
}