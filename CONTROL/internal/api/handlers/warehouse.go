// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// WarehouseHandler handles data warehouse operations.
type WarehouseHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewWarehouseHandler creates a new warehouse handler.
func NewWarehouseHandler(db *sqlx.DB, logger *zap.Logger) *WarehouseHandler {
	return &WarehouseHandler{
		db:     db,
		logger: logger,
	}
}

// RecordPayload represents incoming records from PROCESSING module.
type RecordPayload struct {
	Records   []RecordData `json:"records"`
	Source    string       `json:"source"`
	Timestamp time.Time    `json:"timestamp"`
}

// RecordData represents a single record.
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

// ImportRecords imports records from PROCESSING module.
func (h *WarehouseHandler) ImportRecords(c *gin.Context) {
	var payload RecordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("importing records",
		zap.String("source", payload.Source),
		zap.Int("count", len(payload.Records)),
	)

	// Insert records
	query := `
		INSERT INTO sra_records (
			accession, title, platform, instrument, library_strategy, library_source,
			library_layout, organism, tax_id, bio_project, bio_sample,
			total_reads, total_bases, avg_length, imported_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (accession) DO UPDATE SET
			title = EXCLUDED.title,
			total_reads = EXCLUDED.total_reads,
			total_bases = EXCLUDED.total_bases,
			imported_at = EXCLUDED.imported_at`

	tx, err := h.db.BeginTxx(c.Request.Context(), nil)
	if err != nil {
		h.logger.Error("failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	defer tx.Rollback()

	imported := 0
	for _, record := range payload.Records {
		_, err := tx.ExecContext(c.Request.Context(), query,
			record.Accession, record.Title, record.Platform, record.Instrument,
			record.LibraryStrategy, record.LibrarySource, record.LibraryLayout,
			record.Organism, record.TaxID, record.BioProject, record.BioSample,
			record.TotalReads, record.TotalBases, record.AvgLength, time.Now(),
		)
		if err != nil {
			h.logger.Warn("failed to import record",
				zap.String("accession", record.Accession),
				zap.Error(err),
			)
			continue
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		h.logger.Error("failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.logger.Info("records imported",
		zap.Int("imported", imported),
		zap.Int("total", len(payload.Records)),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":  "records imported",
		"imported": imported,
		"total":    len(payload.Records),
	})
}

// SearchRecords searches for records in the warehouse.
func (h *WarehouseHandler) SearchRecords(c *gin.Context) {
	organism := c.Query("organism")
	platform := c.Query("platform")
	strategy := c.Query("strategy")
	limit := 100

	query := `SELECT * FROM sra_records WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if organism != "" {
		query += ` AND organism ILIKE $` + string(rune('0'+argNum))
		args = append(args, "%"+organism+"%")
		argNum++
	}
	if platform != "" {
		query += ` AND platform ILIKE $` + string(rune('0'+argNum))
		args = append(args, "%"+platform+"%")
		argNum++
	}
	if strategy != "" {
		query += ` AND library_strategy = $` + string(rune('0'+argNum))
		args = append(args, strategy)
		argNum++
	}

	query += ` ORDER BY imported_at DESC LIMIT $` + string(rune('0'+argNum))
	args = append(args, limit)

	var records []struct {
		Accession       string    `db:"accession" json:"accession"`
		Title           string    `db:"title" json:"title"`
		Platform        string    `db:"platform" json:"platform"`
		Organism        string    `db:"organism" json:"organism"`
		LibraryStrategy string    `db:"library_strategy" json:"library_strategy"`
		TotalReads      int64     `db:"total_reads" json:"total_reads"`
		ImportedAt      time.Time `db:"imported_at" json:"imported_at"`
	}

	if err := h.db.SelectContext(c.Request.Context(), &records, query, args...); err != nil {
		h.logger.Error("failed to search records", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   len(records),
	})
}

// GetStats returns warehouse statistics.
func (h *WarehouseHandler) GetStats(c *gin.Context) {
	var stats struct {
		TotalRecords    int64 `db:"total_records" json:"total_records"`
		TotalOrganisms  int64 `db:"total_organisms" json:"total_organisms"`
		TotalBases      int64 `db:"total_bases" json:"total_bases"`
		TotalReads      int64 `db:"total_reads" json:"total_reads"`
	}

	query := `
		SELECT 
			COUNT(*) as total_records,
			COUNT(DISTINCT organism) as total_organisms,
			COALESCE(SUM(total_bases), 0) as total_bases,
			COALESCE(SUM(total_reads), 0) as total_reads
		FROM sra_records`

	if err := h.db.GetContext(c.Request.Context(), &stats, query); err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
