// Package etl provides ETL functionality.
package etl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"go.uber.org/zap"
)

// Loader handles loading data to the CONTROL module's Data Warehouse.
type Loader struct {
	config config.ControlAPIConfig
	client *http.Client
	logger *zap.Logger
}

// NewLoader creates a new Loader.
func NewLoader(cfg config.ControlAPIConfig, logger *zap.Logger) *Loader {
	return &Loader{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}
}

// LoadBatch sends a batch of records to the CONTROL API.
func (l *Loader) LoadBatch(ctx context.Context, records []*TransformedRecord) error {
	l.logger.Debug("loading batch", zap.Int("size", len(records)))

	// Prepare payload
	payload := l.preparePayload(records)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	// Send to CONTROL API
	url := fmt.Sprintf("%s/api/v1/warehouse/records", l.config.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.config.APIKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// RecordPayload represents the payload sent to CONTROL API.
type RecordPayload struct {
	Records   []RecordData `json:"records"`
	Source    string       `json:"source"`
	Timestamp time.Time    `json:"timestamp"`
}

// RecordData represents a single record in the payload.
type RecordData struct {
	Accession       string            `json:"accession"`
	Title           string            `json:"title"`
	Platform        string            `json:"platform"`
	Instrument      string            `json:"instrument"`
	LibraryName     string            `json:"library_name"`
	LibraryStrategy string            `json:"library_strategy"`
	LibrarySource   string            `json:"library_source"`
	LibraryLayout   string            `json:"library_layout"`
	Organism        string            `json:"organism"`
	TaxID           string            `json:"tax_id"`
	BioProject      string            `json:"bio_project"`
	BioSample       string            `json:"bio_sample"`
	SampleName      string            `json:"sample_name"`
	TotalReads      int64             `json:"total_reads"`
	TotalBases      int64             `json:"total_bases"`
	AvgLength       int               `json:"avg_length"`
	Metadata        map[string]string `json:"metadata"`
}

// preparePayload converts transformed records to API payload format.
func (l *Loader) preparePayload(records []*TransformedRecord) *RecordPayload {
	data := make([]RecordData, 0, len(records))

	for _, tr := range records {
		r := tr.Original
		data = append(data, RecordData{
			Accession:       r.Accession,
			Title:           r.Title,
			Platform:        r.Platform,
			Instrument:      r.Instrument,
			LibraryName:     r.LibraryName,
			LibraryStrategy: r.LibraryStrategy,
			LibrarySource:   r.LibrarySource,
			LibraryLayout:   r.LibraryLayout,
			Organism:        r.Organism,
			TaxID:           r.TaxID,
			BioProject:      r.BioProject,
			BioSample:       r.BioSample,
			SampleName:      r.SampleName,
			TotalReads:      r.TotalReads,
			TotalBases:      r.TotalBases,
			AvgLength:       r.AvgLength,
			Metadata:        tr.Metadata,
		})
	}

	return &RecordPayload{
		Records:   data,
		Source:    "PROCESSING",
		Timestamp: time.Now().UTC(),
	}
}

// HealthCheck checks if the CONTROL API is available.
func (l *Loader) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", l.config.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CONTROL API unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
