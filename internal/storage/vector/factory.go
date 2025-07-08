package vector

import (
	"fmt"

	"github.com/anunay/mentis/internal/config"
	"github.com/anunay/mentis/internal/core/ports"
	"github.com/anunay/mentis/internal/storage/vector/qdrant"
	qdrant_client "github.com/qdrant/go-client/qdrant"
)

// Provider represents the vector database provider
type Provider string

const (
	ProviderQdrant   Provider = "qdrant"
	ProviderPinecone Provider = "pinecone" // Future implementation
	ProviderWeaviate Provider = "weaviate" // Future implementation
	ProviderMemory   Provider = "memory"   // Future implementation for testing
)

// NewVectorRepository creates a vector repository based on the configured provider
func NewVectorRepository(cfg *config.VectorConfig) (ports.VectorRepository, error) {
	provider := Provider(cfg.Provider)
	
	switch provider {
	case ProviderQdrant:
		return newQdrantRepository(cfg.Qdrant)
	case ProviderPinecone:
		return nil, fmt.Errorf("pinecone provider not yet implemented")
	case ProviderWeaviate:
		return nil, fmt.Errorf("weaviate provider not yet implemented")
	case ProviderMemory:
		return nil, fmt.Errorf("memory provider not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported vector provider: %s", provider)
	}
}

// newQdrantRepository creates a Qdrant-specific vector repository
func newQdrantRepository(cfg config.QdrantConfig) (ports.VectorRepository, error) {
	// Create Qdrant client
	client, err := qdrant_client.NewClient(&qdrant_client.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		APIKey: cfg.APIKey,
		UseTLS: cfg.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}
	
	// Create repository
	repo := qdrant.NewRepository(client, cfg.Collection)
	return repo, nil
}

// GetSupportedProviders returns a list of supported vector providers
func GetSupportedProviders() []Provider {
	return []Provider{
		ProviderQdrant,
		// Future providers will be added here as they're implemented
	}
}

// IsProviderSupported checks if a provider is supported
func IsProviderSupported(provider string) bool {
	for _, p := range GetSupportedProviders() {
		if string(p) == provider {
			return true
		}
	}
	return false
}