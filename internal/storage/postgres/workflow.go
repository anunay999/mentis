package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/google/uuid"
)

type WorkflowRepository struct {
	db *sql.DB
}

func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) StoreSession(ctx context.Context, session *domain.WorkflowSession) error {
	contextJSON, err := json.Marshal(session.Context)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO workflow_sessions (id, goal, context, created_at, updated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			goal = EXCLUDED.goal,
			context = EXCLUDED.context,
			updated_at = EXCLUDED.updated_at,
			status = EXCLUDED.status
	`

	_, err = r.db.ExecContext(ctx, query,
		session.ID,
		session.Goal,
		contextJSON,
		session.CreatedAt,
		session.UpdatedAt,
		session.Status,
	)
	return err
}

func (r *WorkflowRepository) GetSession(ctx context.Context, id uuid.UUID) (*domain.WorkflowSession, error) {
	query := `
		SELECT id, goal, context, created_at, updated_at, status
		FROM workflow_sessions
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanSession(row)
}

func (r *WorkflowRepository) UpdateSession(ctx context.Context, session *domain.WorkflowSession) error {
	contextJSON, err := json.Marshal(session.Context)
	if err != nil {
		return err
	}

	query := `
		UPDATE workflow_sessions
		SET goal = $2, context = $3, updated_at = $4, status = $5
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		session.ID,
		session.Goal,
		contextJSON,
		time.Now(),
		session.Status,
	)
	return err
}

func (r *WorkflowRepository) StoreStep(ctx context.Context, step *domain.WorkflowStep) error {
	metadataJSON, err := json.Marshal(step.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO workflow_steps (id, session_id, step_type, artifact_id, input_hash, output_hash, metadata, created_at, completed_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			artifact_id = EXCLUDED.artifact_id,
			output_hash = EXCLUDED.output_hash,
			metadata = EXCLUDED.metadata,
			completed_at = EXCLUDED.completed_at,
			status = EXCLUDED.status
	`

	_, err = r.db.ExecContext(ctx, query,
		step.ID,
		step.SessionID,
		step.StepType,
		step.ArtifactID,
		step.InputHash,
		step.OutputHash,
		metadataJSON,
		step.CreatedAt,
		step.CompletedAt,
		step.Status,
	)
	return err
}

func (r *WorkflowRepository) GetStep(ctx context.Context, id uuid.UUID) (*domain.WorkflowStep, error) {
	query := `
		SELECT id, session_id, step_type, artifact_id, input_hash, output_hash, metadata, created_at, completed_at, status
		FROM workflow_steps
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanStep(row)
}

func (r *WorkflowRepository) UpdateStep(ctx context.Context, step *domain.WorkflowStep) error {
	metadataJSON, err := json.Marshal(step.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE workflow_steps
		SET artifact_id = $2, output_hash = $3, metadata = $4, completed_at = $5, status = $6
		WHERE id = $1
	`

	_, err = r.db.ExecContext(ctx, query,
		step.ID,
		step.ArtifactID,
		step.OutputHash,
		metadataJSON,
		step.CompletedAt,
		step.Status,
	)
	return err
}

func (r *WorkflowRepository) GetStepsBySession(ctx context.Context, sessionID uuid.UUID) ([]*domain.WorkflowStep, error) {
	query := `
		SELECT id, session_id, step_type, artifact_id, input_hash, output_hash, metadata, created_at, completed_at, status
		FROM workflow_steps
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.WorkflowStep
	for rows.Next() {
		step, err := r.scanStep(rows)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return steps, rows.Err()
}

func (r *WorkflowRepository) FindStepByInputHash(ctx context.Context, stepType, inputHash string) (*domain.WorkflowStep, error) {
	query := `
		SELECT id, session_id, step_type, artifact_id, input_hash, output_hash, metadata, created_at, completed_at, status
		FROM workflow_steps
		WHERE step_type = $1 AND input_hash = $2 AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, stepType, inputHash)
	return r.scanStep(row)
}

func (r *WorkflowRepository) FindSimilarSteps(ctx context.Context, stepType string, embedding []float32, topK int) ([]domain.WorkflowStepResult, error) {
	// This is a simplified implementation - in production, you'd want to use pgvector
	// or integrate with the vector database for similarity search
	query := `
		SELECT id, session_id, step_type, artifact_id, input_hash, output_hash, metadata, created_at, completed_at, status
		FROM workflow_steps
		WHERE step_type = $1 AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, stepType, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.WorkflowStepResult
	for rows.Next() {
		step, err := r.scanStep(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, domain.WorkflowStepResult{
			Step:  step,
			Score: 1.0, // Placeholder - actual similarity scoring would be done by vector DB
		})
	}

	return results, rows.Err()
}

func (r *WorkflowRepository) scanSession(row interface {
	Scan(dest ...interface{}) error
}) (*domain.WorkflowSession, error) {
	var session domain.WorkflowSession
	var contextJSON []byte

	err := row.Scan(
		&session.ID,
		&session.Goal,
		&contextJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(contextJSON, &session.Context); err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *WorkflowRepository) scanStep(row interface {
	Scan(dest ...interface{}) error
}) (*domain.WorkflowStep, error) {
	var step domain.WorkflowStep
	var metadataJSON []byte
	var artifactID sql.NullString

	err := row.Scan(
		&step.ID,
		&step.SessionID,
		&step.StepType,
		&artifactID,
		&step.InputHash,
		&step.OutputHash,
		&metadataJSON,
		&step.CreatedAt,
		&step.CompletedAt,
		&step.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if artifactID.Valid {
		id, err := uuid.Parse(artifactID.String)
		if err != nil {
			return nil, err
		}
		step.ArtifactID = id
	}

	if err := json.Unmarshal(metadataJSON, &step.Metadata); err != nil {
		return nil, err
	}

	return &step, nil
}