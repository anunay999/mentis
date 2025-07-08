package ports

import (
	"context"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/google/uuid"
)

type WorkflowRepository interface {
	StoreSession(ctx context.Context, session *domain.WorkflowSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*domain.WorkflowSession, error)
	UpdateSession(ctx context.Context, session *domain.WorkflowSession) error
	StoreStep(ctx context.Context, step *domain.WorkflowStep) error
	GetStep(ctx context.Context, id uuid.UUID) (*domain.WorkflowStep, error)
	UpdateStep(ctx context.Context, step *domain.WorkflowStep) error
	GetStepsBySession(ctx context.Context, sessionID uuid.UUID) ([]*domain.WorkflowStep, error)
	FindStepByInputHash(ctx context.Context, stepType, inputHash string) (*domain.WorkflowStep, error)
	FindSimilarSteps(ctx context.Context, stepType string, embedding []float32, topK int) ([]domain.WorkflowStepResult, error)
}

type WorkflowService interface {
	CreateSession(ctx context.Context, goal string, context map[string]interface{}) (*domain.WorkflowSession, error)
	GetSession(ctx context.Context, id uuid.UUID) (*domain.WorkflowSession, error)
	ExecuteStep(ctx context.Context, req *domain.WorkflowStepRequest) (*domain.WorkflowStepResponse, error)
	LookupStep(ctx context.Context, req *domain.WorkflowLookupRequest) (*domain.WorkflowLookupResponse, error)
	CompleteSession(ctx context.Context, sessionID uuid.UUID) error
	FailSession(ctx context.Context, sessionID uuid.UUID, reason string) error
}

type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

type HashService interface {
	ComputeContentHash(content []byte) string
	ComputeInputHash(input interface{}) string
}