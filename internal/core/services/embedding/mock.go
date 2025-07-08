package embedding

import (
	"context"
	"crypto/sha256"
	"math"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return p.createEmbedding(text), nil
}

func (p *MockProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = p.createEmbedding(text)
	}
	return embeddings, nil
}

func (p *MockProvider) GetDimensions() int {
	return 1536
}

func (p *MockProvider) GetModelName() string {
	return "mock-embedding"
}

func (p *MockProvider) createEmbedding(text string) []float32 {
	const embeddingSize = 1536
	
	// Normalize text
	text = strings.ToLower(strings.TrimSpace(text))
	
	// Create hash for deterministic results
	hash := sha256.Sum256([]byte(text))
	
	embedding := make([]float32, embeddingSize)
	
	// Use text characteristics to create varied embeddings
	textLen := len(text)
	wordCount := len(strings.Fields(text))
	
	for i := 0; i < embeddingSize; i++ {
		// Combine hash bytes with text characteristics
		hashIndex := i % len(hash)
		
		// Create a value based on hash, position, and text features
		value := float64(hash[hashIndex]) / 255.0
		
		// Add some variation based on text characteristics
		if i < textLen {
			value += float64(text[i%textLen]) / 255.0
		}
		
		// Add word count influence
		value += float64(wordCount) / 1000.0
		
		// Add positional influence
		value += math.Sin(float64(i) * 0.1)
		
		// Normalize to [-1, 1] range
		value = (value - 1.0) / 2.0
		
		embedding[i] = float32(value)
	}
	
	// L2 normalize the embedding
	p.normalizeEmbedding(embedding)
	
	return embedding
}

func (p *MockProvider) normalizeEmbedding(embedding []float32) {
	var sum float32
	for _, val := range embedding {
		sum += val * val
	}
	
	norm := float32(math.Sqrt(float64(sum)))
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}
}