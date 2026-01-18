-- Create SRA records table (Data Warehouse)
CREATE TABLE IF NOT EXISTS sra_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    accession VARCHAR(50) UNIQUE NOT NULL,
    title TEXT,
    platform VARCHAR(100),
    instrument VARCHAR(100),
    library_strategy VARCHAR(50),
    library_source VARCHAR(50),
    library_layout VARCHAR(20),
    organism VARCHAR(255),
    tax_id VARCHAR(20),
    bio_project VARCHAR(50),
    bio_sample VARCHAR(50),
    total_reads BIGINT DEFAULT 0,
    total_bases BIGINT DEFAULT 0,
    avg_length INTEGER DEFAULT 0,
    imported_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_sra_records_accession ON sra_records(accession);
CREATE INDEX IF NOT EXISTS idx_sra_records_organism ON sra_records(organism);
CREATE INDEX IF NOT EXISTS idx_sra_records_platform ON sra_records(platform);
CREATE INDEX IF NOT EXISTS idx_sra_records_library_strategy ON sra_records(library_strategy);
CREATE INDEX IF NOT EXISTS idx_sra_records_bio_project ON sra_records(bio_project);
CREATE INDEX IF NOT EXISTS idx_sra_records_imported_at ON sra_records(imported_at DESC);

-- Full text search index
CREATE INDEX IF NOT EXISTS idx_sra_records_fts ON sra_records USING gin(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(organism, '')));
