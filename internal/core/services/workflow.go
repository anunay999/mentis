package services

import (
	"context"
	"fmt"
	"time"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/anunay/mentis/internal/core/ports"
	"github.com/google/uuid"
)

type WorkflowService struct {
	workflowRepo    ports.WorkflowRepository
	artifactRepo    ports.ArtifactRepository
	vectorRepo      ports.VectorRepository
	embeddingService ports.EmbeddingService
	hashService     ports.HashService
}

func NewWorkflowService(
	workflowRepo ports.WorkflowRepository,
	artifactRepo ports.ArtifactRepository,
	vectorRepo ports.VectorRepository,
	embeddingService ports.EmbeddingService,
	hashService ports.HashService,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo:    workflowRepo,
		artifactRepo:    artifactRepo,
		vectorRepo:      vectorRepo,
		embeddingService: embeddingService,
		hashService:     hashService,
	}
}

func (s *WorkflowService) CreateSession(ctx context.Context, goal string, sessionContext map[string]interface{}) (*domain.WorkflowSession, error) {
	session := &domain.WorkflowSession{
		ID:        uuid.New(),
		Goal:      goal,
		Context:   sessionContext,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    domain.SessionActive,
	}

	if err := s.workflowRepo.StoreSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return session, nil
}

func (s *WorkflowService) GetSession(ctx context.Context, id uuid.UUID) (*domain.WorkflowSession, error) {
	session, err := s.workflowRepo.GetSession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Load steps
	steps, err := s.workflowRepo.GetStepsBySession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps: %w", err)
	}

	session.Steps = make([]domain.WorkflowStep, len(steps))
	for i, step := range steps {
		session.Steps[i] = *step
	}

	return session, nil
}

func (s *WorkflowService) ExecuteStep(ctx context.Context, req *domain.WorkflowStepRequest) (*domain.WorkflowStepResponse, error) {
	// Compute input hash
	inputHash := s.hashService.ComputeInputHash(req.Input)

	// Check if we have a cached result for this step
	cachedStep, err := s.workflowRepo.FindStepByInputHash(ctx, req.StepType, inputHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check cached step: %w", err)
	}

	if cachedStep != nil {
		// Return cached result
		artifact, err := s.artifactRepo.GetByID(ctx, cachedStep.ArtifactID)
		if err != nil {
			return nil, fmt.Errorf("failed to get cached artifact: %w", err)
		}

		return &domain.WorkflowStepResponse{
			Step:     cachedStep,
			Artifact: artifact,
			Cached:   true,
		}, nil
	}

	// Create new step
	step := &domain.WorkflowStep{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		StepType:  req.StepType,
		InputHash: inputHash,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		Status:    domain.StepRunning,
	}

	if err := s.workflowRepo.StoreStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to store step: %w", err)
	}

	// For now, we'll simulate step execution
	// In production, this would call the actual step processor
	artifact, err := s.simulateStepExecution(ctx, step, req.Input)
	if err != nil {
		step.Status = domain.StepFailed
		s.workflowRepo.UpdateStep(ctx, step)
		return nil, fmt.Errorf("failed to execute step: %w", err)
	}

	// Store the result artifact
	if err := s.artifactRepo.Store(ctx, artifact); err != nil {
		return nil, fmt.Errorf("failed to store artifact: %w", err)
	}

	// Store vector if embedding is available
	if len(artifact.Embedding) > 0 {
		if err := s.vectorRepo.Store(ctx, artifact.ID, artifact.Embedding, artifact.Metadata); err != nil {
			return nil, fmt.Errorf("failed to store vector: %w", err)
		}
	}

	// Update step
	step.ArtifactID = artifact.ID
	step.OutputHash = artifact.ContentHash
	step.Status = domain.StepCompleted
	now := time.Now()
	step.CompletedAt = &now

	if err := s.workflowRepo.UpdateStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	return &domain.WorkflowStepResponse{
		Step:     step,
		Artifact: artifact,
		Cached:   false,
	}, nil
}

func (s *WorkflowService) LookupStep(ctx context.Context, req *domain.WorkflowLookupRequest) (*domain.WorkflowLookupResponse, error) {
	// Generate embedding for the input
	inputText := fmt.Sprintf("%v", req.Input)
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, inputText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search for similar steps
	results, err := s.workflowRepo.FindSimilarSteps(ctx, req.StepType, embedding, req.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar steps: %w", err)
	}

	// Enrich with artifact data
	for i, result := range results {
		if result.Step.ArtifactID != uuid.Nil {
			artifact, err := s.artifactRepo.GetByID(ctx, result.Step.ArtifactID)
			if err == nil {
				results[i].Artifact = artifact
			}
		}
	}

	return &domain.WorkflowLookupResponse{
		Results: results,
	}, nil
}

func (s *WorkflowService) CompleteSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.workflowRepo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.Status = domain.SessionCompleted
	session.UpdatedAt = time.Now()

	return s.workflowRepo.UpdateSession(ctx, session)
}

func (s *WorkflowService) FailSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	session, err := s.workflowRepo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.Status = domain.SessionFailed
	session.UpdatedAt = time.Now()
	if session.Context == nil {
		session.Context = make(map[string]interface{})
	}
	session.Context["failure_reason"] = reason

	return s.workflowRepo.UpdateSession(ctx, session)
}

// simulateStepExecution simulates the execution of a workflow step
// In production, this would be replaced with actual step processors
func (s *WorkflowService) simulateStepExecution(ctx context.Context, step *domain.WorkflowStep, input interface{}) (*domain.Artifact, error) {
	// Create a mock artifact based on the step type
	content := fmt.Sprintf("Result of %s step with input: %v", step.StepType, input)
	contentBytes := []byte(content)

	// Generate embedding
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Determine artifact type based on step type
	var artifactType domain.ArtifactType
	switch step.StepType {
	case "scrape":
		artifactType = domain.RAW
	case "process", "embed":
		artifactType = domain.DERIVED
	case "reason":
		artifactType = domain.REASONING
	case "answer":
		artifactType = domain.ANSWER
	default:
		artifactType = domain.DERIVED
	}

	artifact := &domain.Artifact{
		ID:          uuid.New(),
		Type:        artifactType,
		ContentHash: s.hashService.ComputeContentHash(contentBytes),
		Content:     contentBytes,
		Embedding:   embedding,
		Metadata: map[string]interface{}{
			"step_type":  step.StepType,
			"step_id":    step.ID.String(),
			"session_id": step.SessionID.String(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Stale:     false,
	}

	return artifact, nil
}