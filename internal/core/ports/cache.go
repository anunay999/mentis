package ports

import (
	"context"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/google/uuid"
)

type ArtifactRepository interface {
	Store(ctx context.Context, artifact *domain.Artifact) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error)
	GetByContentHash(ctx context.Context, hash string) (*domain.Artifact, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Artifact, error)
	Update(ctx context.Context, artifact *domain.Artifact) error
	Delete(ctx context.Context, id uuid.UUID) error
	StoreDependency(ctx context.Context, parentID, childID uuid.UUID) error
	GetDependencies(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, error)
	GetDependents(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, error)
	MarkStale(ctx context.Context, artifactID uuid.UUID) error
	MarkStaleBySourceURL(ctx context.Context, sourceURL string) error
}

type VectorRepository interface {
	Store(ctx context.Context, id uuid.UUID, embedding []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, query []float32, topK int, minScore float32, filter map[string]interface{}) ([]domain.LookupResult, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, embedding []float32, metadata map[string]interface{}) error
}

type CacheService interface {
	Publish(ctx context.Context, artifacts []domain.Artifact) (*domain.PublishResponse, error)
	Lookup(ctx context.Context, options domain.LookupOptions) (*domain.LookupResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Invalidate(ctx context.Context, sourceURL string) error
}