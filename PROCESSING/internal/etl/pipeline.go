// Package etl provides Extract, Transform, Load pipeline functionality.
package etl

import (
	"context"
	"fmt"
	"sync"

	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"github.com/guidiju-50/pandora/PROCESSING/internal/models"
	"github.com/guidiju-50/pandora/PROCESSING/internal/scraper"
	"go.uber.org/zap"
)

// Pipeline represents an ETL pipeline for processing biological data.
type Pipeline struct {
	config     config.ETLConfig
	scraper    *scraper.NCBIScraper
	loader     *Loader
	logger     *zap.Logger
	workerPool chan struct{}
}

// NewPipeline creates a new ETL pipeline.
func NewPipeline(cfg config.ETLConfig, scr *scraper.NCBIScraper, loader *Loader, logger *zap.Logger) *Pipeline {
	return &Pipeline{
		config:     cfg,
		scraper:    scr,
		loader:     loader,
		logger:     logger,
		workerPool: make(chan struct{}, cfg.WorkerCount),
	}
}

// ExtractResult holds the result of an extraction operation.
type ExtractResult struct {
	Records []*models.SRARecord
	Errors  []error
}

// TransformResult holds the result of a transformation operation.
type TransformResult struct {
	Records []*TransformedRecord
	Errors  []error
}

// TransformedRecord represents a record after transformation.
type TransformedRecord struct {
	Original    *models.SRARecord
	Validated   bool
	Normalized  bool
	Enriched    bool
	Metadata    map[string]string
}

// Extract fetches data from the source database.
func (p *Pipeline) Extract(ctx context.Context, query string, maxResults int) (*ExtractResult, error) {
	p.logger.Info("starting extraction",
		zap.String("query", query),
		zap.Int("max_results", maxResults),
	)

	records, err := p.scraper.SearchAndFetch(ctx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	result := &ExtractResult{
		Records: records,
	}

	p.logger.Info("extraction completed",
		zap.Int("records_extracted", len(records)),
	)

	return result, nil
}

// ExtractByAccessions fetches data for specific accessions.
func (p *Pipeline) ExtractByAccessions(ctx context.Context, accessions []string) (*ExtractResult, error) {
	p.logger.Info("extracting by accessions",
		zap.Int("count", len(accessions)),
	)

	var (
		records []*models.SRARecord
		errors  []error
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, acc := range accessions {
		wg.Add(1)
		go func(accession string) {
			defer wg.Done()

			// Acquire worker slot
			p.workerPool <- struct{}{}
			defer func() { <-p.workerPool }()

			record, err := p.scraper.GetRunInfo(ctx, accession)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, fmt.Errorf("accession %s: %w", accession, err))
				return
			}
			records = append(records, record)
		}(acc)
	}

	wg.Wait()

	return &ExtractResult{
		Records: records,
		Errors:  errors,
	}, nil
}

// Transform applies transformations to extracted records.
func (p *Pipeline) Transform(ctx context.Context, extracted *ExtractResult) (*TransformResult, error) {
	p.logger.Info("starting transformation",
		zap.Int("input_records", len(extracted.Records)),
	)

	transformed := make([]*TransformedRecord, 0, len(extracted.Records))
	var errors []error

	for _, record := range extracted.Records {
		tr, err := p.transformRecord(ctx, record)
		if err != nil {
			errors = append(errors, fmt.Errorf("transform %s: %w", record.Accession, err))
			continue
		}
		transformed = append(transformed, tr)
	}

	result := &TransformResult{
		Records: transformed,
		Errors:  errors,
	}

	p.logger.Info("transformation completed",
		zap.Int("transformed", len(transformed)),
		zap.Int("errors", len(errors)),
	)

	return result, nil
}

// transformRecord applies transformations to a single record.
func (p *Pipeline) transformRecord(ctx context.Context, record *models.SRARecord) (*TransformedRecord, error) {
	tr := &TransformedRecord{
		Original: record,
		Metadata: make(map[string]string),
	}

	// Validate record
	if err := p.validateRecord(record); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	tr.Validated = true

	// Normalize fields
	p.normalizeRecord(record)
	tr.Normalized = true

	// Enrich with additional metadata
	p.enrichRecord(ctx, tr)
	tr.Enriched = true

	return tr, nil
}

// validateRecord validates a record's required fields.
func (p *Pipeline) validateRecord(record *models.SRARecord) error {
	if record.Accession == "" {
		return fmt.Errorf("missing accession")
	}
	if record.Organism == "" {
		return fmt.Errorf("missing organism")
	}
	return nil
}

// normalizeRecord normalizes record fields.
func (p *Pipeline) normalizeRecord(record *models.SRARecord) {
	// Normalize library layout
	switch record.LibraryLayout {
	case "PAIRED", "paired":
		record.LibraryLayout = "PAIRED"
	case "SINGLE", "single":
		record.LibraryLayout = "SINGLE"
	}

	// Normalize platform
	record.Platform = normalizeString(record.Platform)
	record.Instrument = normalizeString(record.Instrument)
}

// enrichRecord adds additional metadata to a record.
func (p *Pipeline) enrichRecord(ctx context.Context, tr *TransformedRecord) {
	record := tr.Original

	// Add computed metadata
	tr.Metadata["is_paired"] = fmt.Sprintf("%v", record.LibraryLayout == "PAIRED")
	tr.Metadata["is_rnaseq"] = fmt.Sprintf("%v", record.LibraryStrategy == "RNA-Seq")

	// Calculate coverage estimate if genome size is known
	if record.TotalBases > 0 {
		tr.Metadata["total_gigabases"] = fmt.Sprintf("%.2f", float64(record.TotalBases)/1e9)
	}
}

// Load loads transformed records to the data warehouse.
func (p *Pipeline) Load(ctx context.Context, transformed *TransformResult) error {
	p.logger.Info("starting load",
		zap.Int("records", len(transformed.Records)),
	)

	// Process in batches
	batchSize := p.config.BatchSize
	records := transformed.Records

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := p.loader.LoadBatch(ctx, batch); err != nil {
			return fmt.Errorf("loading batch %d-%d: %w", i, end, err)
		}

		p.logger.Debug("batch loaded",
			zap.Int("start", i),
			zap.Int("end", end),
		)
	}

	p.logger.Info("load completed",
		zap.Int("total_loaded", len(records)),
	)

	return nil
}

// Run executes the complete ETL pipeline.
func (p *Pipeline) Run(ctx context.Context, query string, maxResults int) error {
	// Extract
	extracted, err := p.Extract(ctx, query, maxResults)
	if err != nil {
		return fmt.Errorf("extract phase: %w", err)
	}

	// Transform
	transformed, err := p.Transform(ctx, extracted)
	if err != nil {
		return fmt.Errorf("transform phase: %w", err)
	}

	// Load
	if err := p.Load(ctx, transformed); err != nil {
		return fmt.Errorf("load phase: %w", err)
	}

	return nil
}

// normalizeString trims and normalizes a string.
func normalizeString(s string) string {
	// Simple normalization - could be extended
	return s
}
