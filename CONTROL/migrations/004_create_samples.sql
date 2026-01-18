-- Create samples table
CREATE TABLE IF NOT EXISTS samples (
    id UUID PRIMARY KEY,
    experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    accession VARCHAR(50),
    condition VARCHAR(255),
    replicate INTEGER DEFAULT 1,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_samples_experiment_id ON samples(experiment_id);
CREATE INDEX IF NOT EXISTS idx_samples_accession ON samples(accession);
