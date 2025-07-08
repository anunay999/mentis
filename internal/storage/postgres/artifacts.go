package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/google/uuid"
)

type ArtifactRepository struct {
	db *sql.DB
}

func NewArtifactRepository(db *sql.DB) *ArtifactRepository {
	return &ArtifactRepository{db: db}
}

func (r *ArtifactRepository) Store(ctx context.Context, artifact *domain.Artifact) error {
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO artifacts (id, type, content_hash, content, metadata, created_at, updated_at, stale)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			content_hash = EXCLUDED.content_hash,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at,
			stale = EXCLUDED.stale
	`

	_, err = r.db.ExecContext(ctx, query,
		artifact.ID,
		artifact.Type,
		artifact.ContentHash,
		artifact.Content,
		metadataJSON,
		artifact.CreatedAt,
		artifact.UpdatedAt,
		artifact.Stale,
	)
	return err
}

func (r *ArtifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error) {
	query := `
		SELECT id, type, content_hash, content, metadata, created_at, updated_at, stale
		FROM artifacts
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanArtifact(row)
}

func (r *ArtifactRepository) GetByContentHash(ctx context.Context, hash string) (*domain.Artifact, error) {
	query := `
		SELECT id, type, content_hash, content, metadata, created_at, updated_at, stale
		FROM artifacts
		WHERE content_hash = $1
	`

	row := r.db.QueryRowContext(ctx, query, hash)
	return r.scanArtifact(row)
}

func (r *ArtifactRepository) List(ctx context.Context, limit, offset int) ([]*domain.Artifact, error) {
	query := `
		SELECT id, type, content_hash, content, metadata, created_at, updated_at, stale
		FROM artifacts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*domain.Artifact
	for rows.Next() {
		artifact, err := r.scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, rows.Err()
}

func (r *ArtifactRepository) Update(ctx context.Context, artifact *domain.Artifact) error {
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE artifacts
		SET type = $2, content_hash = $3, content = $4, metadata = $5, updated_at = $6, stale = $7
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		artifact.ID,
		artifact.Type,
		artifact.ContentHash,
		artifact.Content,
		metadataJSON,
		time.Now(),
		artifact.Stale,
	)
	return err
}

func (r *ArtifactRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM artifacts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *ArtifactRepository) StoreDependency(ctx context.Context, parentID, childID uuid.UUID) error {
	query := `
		INSERT INTO artifact_dependencies (parent_id, child_id)
		VALUES ($1, $2)
		ON CONFLICT (parent_id, child_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, parentID, childID)
	return err
}

func (r *ArtifactRepository) GetDependencies(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT child_id
		FROM artifact_dependencies
		WHERE parent_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependencies []uuid.UUID
	for rows.Next() {
		var depID uuid.UUID
		if err := rows.Scan(&depID); err != nil {
			return nil, err
		}
		dependencies = append(dependencies, depID)
	}

	return dependencies, rows.Err()
}

func (r *ArtifactRepository) GetDependents(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT parent_id
		FROM artifact_dependencies
		WHERE child_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependents []uuid.UUID
	for rows.Next() {
		var depID uuid.UUID
		if err := rows.Scan(&depID); err != nil {
			return nil, err
		}
		dependents = append(dependents, depID)
	}

	return dependents, rows.Err()
}

func (r *ArtifactRepository) MarkStale(ctx context.Context, artifactID uuid.UUID) error {
	query := `UPDATE artifacts SET stale = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, artifactID)
	return err
}

func (r *ArtifactRepository) MarkStaleBySourceURL(ctx context.Context, sourceURL string) error {
	query := `
		UPDATE artifacts
		SET stale = true, updated_at = NOW()
		WHERE metadata->>'source_url' = $1
	`
	_, err := r.db.ExecContext(ctx, query, sourceURL)
	return err
}

func (r *ArtifactRepository) scanArtifact(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Artifact, error) {
	var artifact domain.Artifact
	var metadataJSON []byte

	err := row.Scan(
		&artifact.ID,
		&artifact.Type,
		&artifact.ContentHash,
		&artifact.Content,
		&metadataJSON,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
		&artifact.Stale,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
		return nil, err
	}

	return &artifact, nil
}