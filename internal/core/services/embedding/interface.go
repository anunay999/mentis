package embedding

import (
	"context"
	"fmt"

	"github.com/anunay/mentis/internal/config"
	"github.com/anunay/mentis/internal/core/ports"
)

type Provider interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	GetDimensions() int
	GetModelName() string
}

type Service struct {
	provider Provider
}

func NewService(cfg config.EmbeddingConfig) (ports.EmbeddingService, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		provider, err = NewOpenAIProvider(cfg.OpenAI)
	case "gemini":
		if cfg.Gemini.APIKey == "" {
			return nil, fmt.Errorf("Gemini API key is required")
		}
		provider, err = NewGeminiProvider(cfg.Gemini)
	case "openai_compatible":
		if cfg.Compatible.BaseURL == "" {
			return nil, fmt.Errorf("Base URL is required for OpenAI-compatible provider")
		}
		provider, err = NewOpenAICompatibleProvider(cfg.Compatible)
	case "mock":
		provider = NewMockProvider()
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create embedding provider: %w", err)
	}

	return &Service{provider: provider}, nil
}

func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return s.provider.GenerateEmbedding(ctx, text)
}

func (s *Service) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	return s.provider.GenerateEmbeddings(ctx, texts)
}

func (s *Service) GetDimensions() int {
	return s.provider.GetDimensions()
}

func (s *Service) GetModelName() string {
	return s.provider.GetModelName()
}