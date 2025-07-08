package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowStep struct {
	ID          uuid.UUID              `json:"id"`
	SessionID   uuid.UUID              `json:"session_id"`
	StepType    string                 `json:"step_type"`
	ArtifactID  uuid.UUID              `json:"artifact_id"`
	InputHash   string                 `json:"input_hash"`
	OutputHash  string                 `json:"output_hash"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at"`
	Status      StepStatus             `json:"status"`
}

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
)

type WorkflowSession struct {
	ID        uuid.UUID              `json:"id"`
	Goal      string                 `json:"goal"`
	Context   map[string]interface{} `json:"context"`
	Steps     []WorkflowStep         `json:"steps"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Status    SessionStatus          `json:"status"`
}

type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
)

type WorkflowStepRequest struct {
	SessionID uuid.UUID              `json:"session_id"`
	StepType  string                 `json:"step_type"`
	Input     interface{}            `json:"input"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type WorkflowStepResponse struct {
	Step     *WorkflowStep `json:"step"`
	Artifact *Artifact     `json:"artifact"`
	Cached   bool          `json:"cached"`
}

type WorkflowLookupRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	StepType  string    `json:"step_type"`
	Input     interface{} `json:"input"`
	TopK      int       `json:"top_k"`
}

type WorkflowLookupResponse struct {
	Results []WorkflowStepResult `json:"results"`
}

type WorkflowStepResult struct {
	Step     *WorkflowStep `json:"step"`
	Artifact *Artifact     `json:"artifact"`
	Score    float32       `json:"score"`
}