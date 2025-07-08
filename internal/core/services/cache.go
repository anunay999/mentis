package services

import (
	"context"
	"fmt"
	"time"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/anunay/mentis/internal/core/ports"
	"github.com/google/uuid"
)

type CacheService struct {
	artifactRepo ports.ArtifactRepository
	vectorRepo   ports.VectorRepository
	hashService  ports.HashService
}

func NewCacheService(
	artifactRepo ports.ArtifactRepository,
	vectorRepo ports.VectorRepository,
	hashService ports.HashService,
) *CacheService {
	return &CacheService{
		artifactRepo: artifactRepo,
		vectorRepo:   vectorRepo,
		hashService:  hashService,
	}
}

func (s *CacheService) Publish(ctx context.Context, artifacts []domain.Artifact) (*domain.PublishResponse, error) {
	var published []uuid.UUID
	var skipped []uuid.UUID

	for _, artifact := range artifacts {
		// Set ID if not provided
		if artifact.ID == uuid.Nil {
			artifact.ID = uuid.New()
		}

		// Set timestamps
		if artifact.CreatedAt.IsZero() {
			artifact.CreatedAt = time.Now()
		}
		artifact.UpdatedAt = time.Now()

		// Compute content hash if not provided
		if artifact.ContentHash == "" {
			artifact.ContentHash = s.hashService.ComputeContentHash(artifact.Content)
		}

		// Check if artifact already exists with same content hash
		existing, err := s.artifactRepo.GetByContentHash(ctx, artifact.ContentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing artifact: %w", err)
		}

		if existing != nil {
			skipped = append(skipped, existing.ID)
			continue
		}

		// Store artifact in database
		if err := s.artifactRepo.Store(ctx, &artifact); err != nil {
			return nil, fmt.Errorf("failed to store artifact: %w", err)
		}

		// Store vector if embedding is provided
		if len(artifact.Embedding) > 0 {
			if err := s.vectorRepo.Store(ctx, artifact.ID, artifact.Embedding, artifact.Metadata); err != nil {
				return nil, fmt.Errorf("failed to store vector: %w", err)
			}
		}

		// Store dependencies
		for _, depID := range artifact.Dependencies {
			if err := s.artifactRepo.StoreDependency(ctx, depID, artifact.ID); err != nil {
				return nil, fmt.Errorf("failed to store dependency: %w", err)
			}
		}

		published = append(published, artifact.ID)
	}

	return &domain.PublishResponse{
		Published: published,
		Skipped:   skipped,
	}, nil
}

func (s *CacheService) Lookup(ctx context.Context, options domain.LookupOptions) (*domain.LookupResponse, error) {
	if options.TopK == 0 {
		options.TopK = 10
	}
	if options.MinScore == 0 {
		options.MinScore = 0.85
	}

	// For now, we'll use a simple text embedding approach
	// In production, you'd use a proper embedding service
	queryEmbedding := s.generateSimpleEmbedding(options.Query)

	// Build filter
	filter := make(map[string]interface{})
	if options.ArtifactType != "" {
		filter["type"] = string(options.ArtifactType)
	}
	if !options.IncludeStale {
		filter["stale"] = false
	}

	// Search vectors
	vectorResults, err := s.vectorRepo.Search(ctx, queryEmbedding, options.TopK, options.MinScore, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Enrich results with full artifact data
	var results []domain.LookupResult
	for _, vr := range vectorResults {
		artifact, err := s.artifactRepo.GetByID(ctx, vr.Artifact.ID)
		if err != nil {
			continue
		}

		if artifact == nil {
			continue
		}

		// Apply content/embedding inclusion options
		if !options.IncludeContent {
			artifact.Content = nil
		}
		if !options.IncludeEmbedding {
			artifact.Embedding = nil
		}

		results = append(results, domain.LookupResult{
			Artifact: artifact,
			Score:    vr.Score,
		})
	}

	return &domain.LookupResponse{
		Results: results,
	}, nil
}

func (s *CacheService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error) {
	return s.artifactRepo.GetByID(ctx, id)
}

func (s *CacheService) Delete(ctx context.Context, id uuid.UUID) error {
	// Delete from vector store
	if err := s.vectorRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	// Delete from artifact store
	if err := s.artifactRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	return nil
}

func (s *CacheService) Invalidate(ctx context.Context, sourceURL string) error {
	// Mark artifacts as stale
	if err := s.artifactRepo.MarkStaleBySourceURL(ctx, sourceURL); err != nil {
		return fmt.Errorf("failed to mark artifacts as stale: %w", err)
	}

	return nil
}

// generateSimpleEmbedding creates a simple embedding for demonstration
// This is kept as a fallback when no embedding service is available
func (s *CacheService) generateSimpleEmbedding(text string) []float32 {
	// This is a placeholder - create a simple hash-based embedding
	hash := s.hashService.ComputeInputHash(text)
	embedding := make([]float32, 1536)
	
	for i := 0; i < len(embedding) && i < len(hash); i++ {
		embedding[i] = float32(hash[i]) / 255.0
	}
	
	return embedding
}