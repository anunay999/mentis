# Mentis Technical Design Document
**Workflow-Aware Semantic Cache for AI Agents**

**Author:** Anunay Aatipamula
**Date:** 2025-07-08  
**Version:** 1.0

---

## 1. System Overview

Mentis is a production-ready semantic cache designed specifically for AI agent workloads. Unlike traditional caches that operate on exact key matches, Mentis understands the semantic relationships between queries and the multi-step nature of agent workflows.

### Core Innovation
**Workflow-Step Granularity**: Instead of caching only final results, Mentis caches every step of agent workflows (scrape → process → embed → reason → answer) enabling unprecedented reuse across similar but non-identical agent sessions.

---

## 2. Architecture

### 2.1 High-Level Architecture

```mermaid
graph TD

    %% Define client agents
    A1["AI Agent (Client)"]
    A2["AI Agent (Client)"]
    A3["AI Agent (Client)"]

    %% Define Mentis Coordinator
    B["Mentis Coordinator (Go Service)"]

    %% Connect agents to coordinator
    A1 -->|"REST API (HTTP)"| B
    A2 -->|"REST API (HTTP)"| B
    A3 -->|"REST API (HTTP)"| B

    %% Define databases and providers
    C["PostgreSQL (Metadata)"]
    D["Qdrant (Vectors)"]
    E["Embedding Providers"]

    %% Coordinator connections
    B --> C
    B --> D
    B --> E

    %% PostgreSQL sub-nodes
    C1[Artifacts]
    C2[Sessions]
    C3[Steps]
    C4[Dependencies]
    C --> C1
    C --> C2
    C --> C3
    C --> C4

    %% Qdrant sub-nodes
    D1["Semantic Search"]
    D2["1536-dim Vectors"]
    D --> D1
    D --> D2

    %% Embedding Providers sub-nodes
    E1[OpenAI]
    E2[Gemini]
    E3[OpenAI-Compatible]
    E --> E1
    E --> E2
    E --> E3
```

### 2.2 Component Architecture

```
internal/
├── api/
│   ├── handlers/          # HTTP request handlers
│   │   ├── cache.go      # Artifact operations
│   │   └── workflow.go   # Workflow operations
│   └── middleware/       # Auth, logging, CORS
├── core/
│   ├── domain/          # Business entities
│   │   ├── artifact.go  # Core artifact types
│   │   └── workflow.go  # Workflow session/step
│   ├── ports/           # Interface contracts
│   │   ├── cache.go     # Cache repository interface
│   │   └── workflow.go  # Workflow service interface
│   └── services/        # Business logic
│       ├── cache.go     # Artifact management
│       ├── workflow.go  # Workflow orchestration
│       └── embedding/   # Multi-provider embeddings
├── storage/
│   ├── postgres/        # Metadata persistence
│   │   ├── artifacts.go # Artifact CRUD
│   │   └── workflow.go  # Workflow persistence
│   └── qdrant/         # Vector operations
│       └── client.go   # gRPC-based client
└── config/             # Configuration management
```

---

## 3. Data Model

### 3.1 Core Entities

#### Artifact
```go
type Artifact struct {
    ID           uuid.UUID              `json:"id"`
    Type         ArtifactType           `json:"type"`        // RAW, DERIVED, REASONING, ANSWER
    ContentHash  string                 `json:"content_hash"` // SHA-256 for deduplication
    Content      []byte                 `json:"content"`     // Actual content
    Embedding    []float32              `json:"embedding"`   // Vector representation
    Dependencies []uuid.UUID            `json:"dependencies"` // Parent artifacts
    Metadata     map[string]interface{} `json:"metadata"`    // Source URLs, costs, etc.
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
    Stale        bool                   `json:"stale"`       // Invalidation flag
}
```

#### Workflow Session
```go
type WorkflowSession struct {
    ID        uuid.UUID              `json:"id"`
    Goal      string                 `json:"goal"`        // High-level objective
    Context   map[string]interface{} `json:"context"`     // Session metadata
    Steps     []WorkflowStep         `json:"steps"`       // Executed steps
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
    Status    SessionStatus          `json:"status"`      // active, completed, failed
}
```

#### Workflow Step
```go
type WorkflowStep struct {
    ID          uuid.UUID              `json:"id"`
    SessionID   uuid.UUID              `json:"session_id"`
    StepType    string                 `json:"step_type"`   // scrape, embed, reason, etc.
    ArtifactID  uuid.UUID              `json:"artifact_id"` // Result artifact
    InputHash   string                 `json:"input_hash"`  // SHA-256 of input
    OutputHash  string                 `json:"output_hash"` // SHA-256 of output
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time              `json:"created_at"`
    CompletedAt *time.Time             `json:"completed_at"`
    Status      StepStatus             `json:"status"`      // pending, running, completed, failed
}
```

### 3.2 Database Schema

#### PostgreSQL Tables
```sql
-- Core artifacts table
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('RAW', 'DERIVED', 'REASONING', 'ANSWER')),
    content_hash CHAR(64) NOT NULL,
    content BYTEA,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    stale BOOLEAN DEFAULT FALSE
);

-- Artifact dependency graph
CREATE TABLE artifact_dependencies (
    parent_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    child_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (parent_id, child_id)
);

-- Workflow sessions
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal TEXT NOT NULL,
    context JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed'))
);

-- Workflow steps with caching optimization
CREATE TABLE workflow_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,
    step_type VARCHAR(100) NOT NULL,
    artifact_id UUID REFERENCES artifacts(id) ON DELETE SET NULL,
    input_hash CHAR(64) NOT NULL,
    output_hash CHAR(64),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

-- Unique index for step-level caching
CREATE UNIQUE INDEX idx_workflow_steps_dedup 
ON workflow_steps(step_type, input_hash) 
WHERE status = 'completed';
```

#### Qdrant Vector Storage
```json
{
  "collection_name": "mentis",
  "vector_config": {
    "size": 1536,  // Configurable based on embedding provider
    "distance": "Cosine"
  },
  "payload_schema": {
    "artifact_id": "string",
    "type": "string",
    "source_url": "string", 
    "stale": "boolean",
    "created_at": "datetime"
  }
}
```

---

## 4. Core Services

### 4.1 Embedding Service

Multi-provider abstraction supporting different embedding APIs:

```go
type Provider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
    GetDimensions() int
    GetModelName() string
}
```

#### Supported Providers

| Provider | Models | Dimensions | Use Case |
|----------|--------|------------|----------|
| **OpenAI** | text-embedding-3-small/large | 1536/3072 | Production, high quality |
| **Gemini** | text-embedding-004 | 768 | Google ecosystem |
| **OpenAI-Compatible** | Various (Ollama, Azure) | 384-1536 | Self-hosted, cost optimization |
| **Mock** | Deterministic hash-based | 1536 | Development, testing |

#### Configuration
```env
# OpenAI
EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=sk-your-key
OPENAI_MODEL=text-embedding-3-small

# Local Ollama
EMBEDDING_PROVIDER=openai_compatible  
EMBEDDING_BASE_URL=http://localhost:11434/v1
EMBEDDING_MODEL=nomic-embed-text
```

### 4.2 Cache Service

Core caching logic with semantic similarity:

```go
type CacheService interface {
    Publish(ctx context.Context, artifacts []Artifact) (*PublishResponse, error)
    Lookup(ctx context.Context, options LookupOptions) (*LookupResponse, error)
    GetByID(ctx context.Context, id uuid.UUID) (*Artifact, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Invalidate(ctx context.Context, sourceURL string) error
}
```

#### Lookup Algorithm
1. **Generate Query Embedding**: Convert text query to vector using configured provider
2. **Vector Search**: Query Qdrant with cosine similarity, min threshold 0.85
3. **Metadata Filtering**: Apply type, staleness, and other filters
4. **Enrich Results**: Fetch full artifact data from PostgreSQL
5. **Rank by Relevance**: Combine semantic score with recency and usage

### 4.3 Workflow Service

Manages agent workflow sessions and step-level caching:

```go
type WorkflowService interface {
    CreateSession(ctx context.Context, goal string, context map[string]interface{}) (*WorkflowSession, error)
    ExecuteStep(ctx context.Context, req *WorkflowStepRequest) (*WorkflowStepResponse, error)
    LookupStep(ctx context.Context, req *WorkflowLookupRequest) (*WorkflowLookupResponse, error)
    CompleteSession(ctx context.Context, sessionID uuid.UUID) error
}
```

#### Step Execution Flow
1. **Input Hashing**: Create SHA-256 hash of step input for exact matching
2. **Cache Check**: Look for completed step with same type + input hash
3. **Semantic Fallback**: If no exact match, search for semantically similar steps
4. **Execute or Reuse**: Either return cached artifact or execute step
5. **Store Result**: Cache new artifact and link to workflow step

---

## 5. API Design

### 5.1 REST Endpoints

#### Cache Operations
```http
POST /v1/cache/publish
GET  /v1/cache/lookup
GET  /v1/cache/artifacts/{id}
DELETE /v1/cache/artifacts/{id}
POST /v1/cache/invalidate
```

#### Workflow Operations  
```http
POST /v1/workflow/sessions
GET  /v1/workflow/sessions/{id}
POST /v1/workflow/steps
POST /v1/workflow/steps/lookup
```

#### Quick Access
```http
GET /v1/lookup?q=query&top_k=5&min_score=0.8
GET /v1/workflow/lookup?session_id=...&step_type=scrape&input=...
```

### 5.2 Example API Usage

#### Publishing an Artifact
```json
POST /v1/cache/publish
{
  "objects": [
    {
      "type": "RAW",
      "content": "base64-encoded-content",
      "metadata": {
        "source_url": "https://example.com/article",
        "content_type": "text/html",
        "scraped_at": "2025-07-08T10:00:00Z"
      }
    }
  ]
}
```

#### Semantic Lookup
```json
GET /v1/lookup?q=Apple M3 performance&top_k=5&min_score=0.85

Response:
{
  "results": [
    {
      "artifact": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "type": "DERIVED", 
        "metadata": {
          "source_url": "https://apple.com/m3-specs"
        },
        "stale": false
      },
      "score": 0.92
    }
  ]
}
```

#### Workflow Step Execution
```json
POST /v1/workflow/steps
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "step_type": "scrape",
  "input": "https://example.com/data",
  "metadata": {
    "timeout": 30,
    "user_agent": "mentis-agent"
  }
}

Response:
{
  "step": {
    "id": "987fcdeb-51a2-43d7-af12-345678901234",
    "status": "completed"
  },
  "artifact": {
    "id": "456e7890-a1b2-3c4d-5e6f-789012345678",
    "type": "RAW"
  },
  "cached": false
}
```

---

## 6. Performance Characteristics

### 6.1 Latency Targets

| Operation | P50 | P95 | P99 |
|-----------|-----|-----|-----|
| **Semantic Lookup** | 50ms | 120ms | 250ms |
| **Exact Cache Hit** | 10ms | 25ms | 50ms |
| **Publish Artifact** | 100ms | 200ms | 400ms |
| **Workflow Step** | 80ms | 150ms | 300ms |

### 6.2 Throughput Capacity

- **Sustained RPS**: 1,000 requests/second
- **Burst RPS**: 5,000 requests/second  
- **Concurrent Workflows**: 10,000+ active sessions
- **Vector Search**: 100M+ vectors with <150ms latency

### 6.3 Storage Scalability

- **Artifacts**: 10M+ artifacts per deployment
- **Vector Dimensions**: 384-3072 (provider dependent)
- **Content Size**: Up to 10MB per artifact
- **Session History**: 90-day retention default

---

## 7. Deployment Architecture

### 7.1 Single-Node Development
```yaml
version: '3.8'
services:
  mentis:
    build: .
    ports: ["8080:8080"]
    environment:
      EMBEDDING_PROVIDER: mock
  
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: mentis
      
  qdrant:
    image: qdrant/qdrant:v1.7.0
    ports: ["6333:6333"]
```

### 7.2 Production Multi-Node
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   mentis-1  │    │   mentis-2  │    │   mentis-3  │
│ (coordinator)│    │ (coordinator)│    │ (coordinator)│
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ PostgreSQL  │    │   Qdrant    │    │  Embedding  │
│  (Primary)  │    │  (Cluster)  │    │  Providers  │
│             │    │             │    │  (External) │
│ Read Replica│    │ 3x Replicas │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
```

### 7.3 Configuration Management

#### Environment Variables
```bash
# Core Services  
DATABASE_URL=postgres://user:pass@host:5432/mentis
QDRANT_URL=http://qdrant-cluster:6333
SERVER_PORT=8080

# Embedding Configuration
EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=text-embedding-3-small

# Operational
LOG_LEVEL=info
QDRANT_COLLECTION=mentis
```

---

## 8. Security & Privacy

### 8.1 Data Protection
- **Encryption at Rest**: AES-256 for PostgreSQL, Qdrant encrypted storage
- **Encryption in Transit**: TLS 1.3 for all HTTP communications
- **Key Management**: Environment-based secrets, support for vault integration

### 8.2 Access Control
- **API Authentication**: Bearer token validation
- **Rate Limiting**: Configurable per-client limits
- **Input Validation**: Strict schema validation, size limits

### 8.3 Privacy Considerations
- **Content Hashing**: SHA-256 for deterministic deduplication
- **Metadata Scrubbing**: Optional PII detection and removal
- **Audit Logging**: Structured logs for compliance requirements

---

## 9. Monitoring & Observability

### 9.1 Key Metrics

#### Business Metrics
```
cache_hit_ratio_total{type="semantic"}     # Target: >70%
cache_hit_ratio_total{type="exact"}        # Target: >90%
tokens_saved_total                         # Cost optimization
workflow_reuse_ratio                       # Agent efficiency
```

#### Technical Metrics
```
http_request_duration_seconds{endpoint}    # Latency tracking
qdrant_search_duration_seconds             # Vector search performance  
postgres_query_duration_seconds            # Database performance
embedding_generation_duration_seconds      # Provider latency
```

#### Error Tracking
```
http_requests_total{status="error"}        # API error rate
embedding_errors_total{provider}           # Provider reliability
cache_invalidation_errors_total            # Data consistency
```

### 9.2 Logging Strategy
```json
{
  "timestamp": "2025-07-08T10:30:45.123Z",
  "level": "info",
  "service": "mentis",
  "operation": "semantic_lookup",
  "query_hash": "abc123...",
  "results_count": 5,
  "duration_ms": 127,
  "cache_hit": true,
  "embedding_provider": "openai"
}
```

### 9.3 Health Checks
- **Service Health**: `/health` endpoint with dependency checks
- **Database Connectivity**: PostgreSQL connection pooling status
- **Vector Store**: Qdrant cluster health and index status
- **Embedding Providers**: API availability and response times

---

## 10. Failure Modes & Recovery

### 10.1 Common Failure Scenarios

#### Embedding Provider Outage
```
Problem: OpenAI API unavailable
Impact: Cannot generate new embeddings
Recovery: 
1. Automatic fallback to mock provider
2. Queue requests for retry when service recovers
3. Alert on provider degradation
```

#### Vector Database Corruption
```
Problem: Qdrant index corruption
Impact: Semantic search returns incorrect results  
Recovery:
1. Rebuild vector index from PostgreSQL artifacts
2. Graceful degradation to exact-match only
3. Health check detects and alerts on poor recall
```

#### PostgreSQL Connection Loss
```
Problem: Database connectivity issues
Impact: Cannot read/write artifact metadata
Recovery:
1. Connection pooling with automatic retry
2. Circuit breaker prevents cascade failures
3. Read-only mode using cached data
```

### 10.2 Data Consistency

#### Artifact-Vector Synchronization
- **Problem**: Vector exists in Qdrant but artifact deleted from PostgreSQL
- **Detection**: Periodic consistency checks via background job
- **Resolution**: Orphaned vector cleanup, artifact recreation from backup

#### Workflow State Corruption  
- **Problem**: Step marked completed but no artifact produced
- **Detection**: Workflow validation during session retrieval
- **Resolution**: Mark step as failed, trigger re-execution

---

## 11. Performance Optimization

### 11.1 Caching Strategies

#### Multi-Level Caching
```
L1: In-Memory (Hot artifacts, 100MB)
L2: Qdrant (Semantic similarity, 10GB)  
L3: PostgreSQL (Full artifact data, 1TB)
L4: Object Store (Large content, unlimited)
```

#### Query Optimization
- **Vector Search**: Pre-filtered by metadata before similarity computation
- **Database Queries**: Indexed on content_hash, artifact_type, created_at
- **Connection Pooling**: 100 max connections, 10 idle connections

### 11.2 Batch Operations
- **Embedding Generation**: Batch up to 100 texts per API call
- **Vector Insertion**: Batch insert 1000 vectors per Qdrant operation
- **Database Writes**: Transaction batching for related artifacts

### 11.3 Resource Management
- **Memory Usage**: LRU eviction for in-memory caches
- **Disk Space**: Automatic cleanup of expired artifacts
- **Network**: Connection reuse for embedding providers

---

## 12. Testing Strategy

### 12.1 Unit Testing
- **Coverage Target**: >90% for core business logic
- **Mock Services**: All external dependencies (embedding APIs, databases)
- **Property Testing**: Semantic similarity invariants

### 12.2 Integration Testing  
- **End-to-End Workflows**: Full agent session simulation
- **Database Consistency**: Artifact and vector synchronization
- **Provider Fallback**: Embedding service degradation scenarios

### 12.3 Performance Testing
- **Load Testing**: 1K RPS sustained for 30 minutes
- **Stress Testing**: 5K RPS burst capacity
- **Memory Profiling**: No memory leaks under load

### 12.4 Semantic Testing
- **Recall Validation**: Known similar queries return expected results
- **Precision Testing**: Dissimilar queries don't match
- **Threshold Tuning**: Optimal similarity cutoff per use case

---

This technical design provides the foundation for a production-ready semantic cache that transforms how AI agents reuse computational work. The modular architecture enables incremental deployment while the semantic-first approach delivers immediate value for agent efficiency.