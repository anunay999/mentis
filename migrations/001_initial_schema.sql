-- Create artifacts table
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

-- Create indexes for artifacts
CREATE INDEX idx_artifacts_content_hash ON artifacts(content_hash);
CREATE INDEX idx_artifacts_type ON artifacts(type);
CREATE INDEX idx_artifacts_created_at ON artifacts(created_at);
CREATE INDEX idx_artifacts_stale ON artifacts(stale);
CREATE INDEX idx_artifacts_metadata_source_url ON artifacts USING GIN ((metadata->>'source_url'));

-- Create artifact_dependencies table for DAG relationships
CREATE TABLE artifact_dependencies (
    parent_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    child_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (parent_id, child_id)
);

-- Create indexes for dependencies
CREATE INDEX idx_artifact_dependencies_parent ON artifact_dependencies(parent_id);
CREATE INDEX idx_artifact_dependencies_child ON artifact_dependencies(child_id);

-- Create workflow_sessions table
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal TEXT NOT NULL,
    context JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed'))
);

-- Create workflow_steps table
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

-- Create indexes for workflow_steps
CREATE INDEX idx_workflow_steps_session_id ON workflow_steps(session_id);
CREATE INDEX idx_workflow_steps_type ON workflow_steps(step_type);
CREATE INDEX idx_workflow_steps_input_hash ON workflow_steps(input_hash);
CREATE INDEX idx_workflow_steps_artifact_id ON workflow_steps(artifact_id);
CREATE INDEX idx_workflow_steps_status ON workflow_steps(status);

-- Create unique constraint on step_type + input_hash for deduplication
CREATE UNIQUE INDEX idx_workflow_steps_dedup ON workflow_steps(step_type, input_hash) WHERE status = 'completed';

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_artifacts_updated_at BEFORE UPDATE ON artifacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workflow_sessions_updated_at BEFORE UPDATE ON workflow_sessions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();