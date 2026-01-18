-- Create results table
CREATE TABLE IF NOT EXISTS results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    data JSONB NOT NULL DEFAULT '{}',
    file_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_results_experiment_id ON results(experiment_id);
CREATE INDEX IF NOT EXISTS idx_results_job_id ON results(job_id);
CREATE INDEX IF NOT EXISTS idx_results_type ON results(type);
