package domain

import (
	"time"

	"github.com/google/uuid"
)

type ArtifactType string

const (
	RAW       ArtifactType = "RAW"
	DERIVED   ArtifactType = "DERIVED"
	REASONING ArtifactType = "REASONING"
	ANSWER    ArtifactType = "ANSWER"
)

type Artifact struct {
	ID           uuid.UUID              `json:"id"`
	Type         ArtifactType           `json:"type"`
	ContentHash  string                 `json:"content_hash"`
	Content      []byte                 `json:"content"`
	Embedding    []float32              `json:"embedding,omitempty"`
	Dependencies []uuid.UUID            `json:"dependencies"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Stale        bool                   `json:"stale"`
}

type LookupResult struct {
	Artifact *Artifact `json:"artifact"`
	Score    float32   `json:"score"`
}

type LookupOptions struct {
	Query           string       `json:"query"`
	TopK            int          `json:"top_k"`
	MinScore        float32      `json:"min_score"`
	ArtifactType    ArtifactType `json:"artifact_type,omitempty"`
	IncludeStale    bool         `json:"include_stale"`
	IncludeContent  bool         `json:"include_content"`
	IncludeEmbedding bool        `json:"include_embedding"`
}

type PublishRequest struct {
	Objects []Artifact `json:"objects"`
}

type PublishResponse struct {
	Published []uuid.UUID `json:"published"`
	Skipped   []uuid.UUID `json:"skipped"`
}

type LookupRequest struct {
	Options LookupOptions `json:"options"`
}

type LookupResponse struct {
	Results []LookupResult `json:"results"`
}