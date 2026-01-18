// Package scraper provides web scraping capabilities for biological databases.
package scraper

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"github.com/guidiju-50/pandora/PROCESSING/internal/models"
	"github.com/guidiju-50/pandora/PROCESSING/pkg/httpclient"
	"go.uber.org/zap"
)

// NCBIScraper handles scraping data from NCBI databases.
type NCBIScraper struct {
	client *httpclient.Client
	config config.NCBIConfig
	logger *zap.Logger
}

// NewNCBIScraper creates a new NCBI scraper.
func NewNCBIScraper(cfg config.NCBIConfig, logger *zap.Logger) *NCBIScraper {
	httpClient := httpclient.NewClient(httpclient.Config{
		Timeout:    cfg.Timeout,
		RateLimit:  cfg.RateLimit,
		MaxRetries: cfg.MaxRetries,
		RetryDelay: cfg.RetryDelay,
	}, logger)

	return &NCBIScraper{
		client: httpClient,
		config: cfg,
		logger: logger,
	}
}

// SearchSRA searches the SRA database for records matching the query.
func (s *NCBIScraper) SearchSRA(ctx context.Context, query string, maxResults int) ([]string, error) {
	s.logger.Info("searching SRA", zap.String("query", query), zap.Int("max_results", maxResults))

	url := fmt.Sprintf("%s/esearch.fcgi?db=sra&term=%s&retmax=%d&usehistory=y",
		s.config.BaseURL, query, maxResults)

	if s.config.APIKey != "" {
		url += "&api_key=" + s.config.APIKey
	}

	data, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("searching SRA: %w", err)
	}

	var result eSearchResult
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	s.logger.Info("SRA search completed",
		zap.Int("count", result.Count),
		zap.Int("returned", len(result.IDList.IDs)),
	)

	return result.IDList.IDs, nil
}

// FetchSRARecord fetches detailed information for an SRA record.
func (s *NCBIScraper) FetchSRARecord(ctx context.Context, id string) (*models.SRARecord, error) {
	s.logger.Info("fetching SRA record", zap.String("id", id))

	url := fmt.Sprintf("%s/efetch.fcgi?db=sra&id=%s&rettype=full&retmode=xml",
		s.config.BaseURL, id)

	if s.config.APIKey != "" {
		url += "&api_key=" + s.config.APIKey
	}

	data, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching SRA record: %w", err)
	}

	record, err := s.parseSRARecord(data)
	if err != nil {
		return nil, fmt.Errorf("parsing SRA record: %w", err)
	}

	return record, nil
}

// FetchSRARecords fetches multiple SRA records.
func (s *NCBIScraper) FetchSRARecords(ctx context.Context, ids []string) ([]*models.SRARecord, error) {
	s.logger.Info("fetching SRA records", zap.Int("count", len(ids)))

	records := make([]*models.SRARecord, 0, len(ids))

	for _, id := range ids {
		record, err := s.FetchSRARecord(ctx, id)
		if err != nil {
			s.logger.Warn("failed to fetch record", zap.String("id", id), zap.Error(err))
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// SearchAndFetch searches SRA and fetches all matching records.
func (s *NCBIScraper) SearchAndFetch(ctx context.Context, query string, maxResults int) ([]*models.SRARecord, error) {
	ids, err := s.SearchSRA(ctx, query, maxResults)
	if err != nil {
		return nil, err
	}

	return s.FetchSRARecords(ctx, ids)
}

// GetRunInfo fetches run info for an SRA accession (SRR/ERR/DRR).
func (s *NCBIScraper) GetRunInfo(ctx context.Context, accession string) (*models.SRARecord, error) {
	s.logger.Info("fetching run info", zap.String("accession", accession))

	// Use SRA Run Selector API
	url := fmt.Sprintf("https://trace.ncbi.nlm.nih.gov/Traces/sra/sra.cgi?save=efetch&db=sra&rettype=runinfo&term=%s",
		accession)

	data, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching run info: %w", err)
	}

	record, err := s.parseRunInfo(data, accession)
	if err != nil {
		return nil, fmt.Errorf("parsing run info: %w", err)
	}

	return record, nil
}

// parseSRARecord parses XML data into an SRARecord.
func (s *NCBIScraper) parseSRARecord(data []byte) (*models.SRARecord, error) {
	var pkg sraExperimentPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	exp := pkg.Experiment
	sample := pkg.Sample
	run := pkg.Run

	record := &models.SRARecord{
		Accession:       exp.Accession,
		Title:           exp.Title,
		Platform:        exp.Platform.InstrumentModel,
		LibraryName:     exp.Design.LibraryDescriptor.LibraryName,
		LibraryStrategy: exp.Design.LibraryDescriptor.LibraryStrategy,
		LibrarySource:   exp.Design.LibraryDescriptor.LibrarySource,
		LibraryLayout:   exp.Design.LibraryDescriptor.LibraryLayout.Layout,
		Organism:        sample.SampleName.ScientificName,
		TaxID:           sample.SampleName.TaxID,
		SampleName:      sample.Alias,
	}

	if len(run.Statistics.Reads) > 0 {
		record.TotalReads = run.Statistics.Reads[0].Count
		record.TotalBases = run.Statistics.Reads[0].Count * int64(run.Statistics.Reads[0].AverageLength)
		record.AvgLength = run.Statistics.Reads[0].AverageLength
	}

	return record, nil
}

// parseRunInfo parses CSV run info into an SRARecord.
func (s *NCBIScraper) parseRunInfo(data []byte, accession string) (*models.SRARecord, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid run info format")
	}

	// Parse header and find columns
	headers := strings.Split(lines[0], ",")
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	// Find the line for our accession
	var values []string
	for _, line := range lines[1:] {
		if strings.Contains(line, accession) {
			values = strings.Split(line, ",")
			break
		}
	}

	if values == nil {
		return nil, fmt.Errorf("accession not found in run info")
	}

	getValue := func(key string) string {
		if idx, ok := headerMap[key]; ok && idx < len(values) {
			return strings.TrimSpace(values[idx])
		}
		return ""
	}

	getInt64 := func(key string) int64 {
		v, _ := strconv.ParseInt(getValue(key), 10, 64)
		return v
	}

	record := &models.SRARecord{
		Accession:       getValue("Run"),
		Title:           getValue("Experiment"),
		Platform:        getValue("Platform"),
		Instrument:      getValue("Model"),
		LibraryName:     getValue("LibraryName"),
		LibraryStrategy: getValue("LibraryStrategy"),
		LibrarySource:   getValue("LibrarySource"),
		LibraryLayout:   getValue("LibraryLayout"),
		Organism:        getValue("ScientificName"),
		TaxID:           getValue("TaxID"),
		BioProject:      getValue("BioProject"),
		BioSample:       getValue("BioSample"),
		SampleName:      getValue("SampleName"),
		TotalReads:      getInt64("spots"),
		TotalBases:      getInt64("bases"),
		AvgLength:       int(getInt64("avgLength")),
	}

	if dateStr := getValue("ReleaseDate"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			record.ReleaseDate = t
		}
	}

	return record, nil
}

// XML structures for NCBI responses

type eSearchResult struct {
	XMLName xml.Name `xml:"eSearchResult"`
	Count   int      `xml:"Count"`
	IDList  struct {
		IDs []string `xml:"Id"`
	} `xml:"IdList"`
	WebEnv   string `xml:"WebEnv"`
	QueryKey string `xml:"QueryKey"`
}

type sraExperimentPackage struct {
	XMLName    xml.Name      `xml:"EXPERIMENT_PACKAGE"`
	Experiment sraExperiment `xml:"EXPERIMENT"`
	Sample     sraSample     `xml:"SAMPLE"`
	Run        sraRun        `xml:"RUN_SET>RUN"`
}

type sraExperiment struct {
	Accession string `xml:"accession,attr"`
	Title     string `xml:"TITLE"`
	Platform  struct {
		InstrumentModel string `xml:"INSTRUMENT_MODEL"`
	} `xml:"PLATFORM>ILLUMINA"`
	Design struct {
		LibraryDescriptor struct {
			LibraryName     string `xml:"LIBRARY_NAME"`
			LibraryStrategy string `xml:"LIBRARY_STRATEGY"`
			LibrarySource   string `xml:"LIBRARY_SOURCE"`
			LibraryLayout   struct {
				Layout string `xml:",innerxml"`
			} `xml:"LIBRARY_LAYOUT"`
		} `xml:"LIBRARY_DESCRIPTOR"`
	} `xml:"DESIGN"`
}

type sraSample struct {
	Accession  string `xml:"accession,attr"`
	Alias      string `xml:"alias,attr"`
	SampleName struct {
		TaxID          string `xml:"TAXON_ID"`
		ScientificName string `xml:"SCIENTIFIC_NAME"`
	} `xml:"SAMPLE_NAME"`
}

type sraRun struct {
	Accession  string `xml:"accession,attr"`
	Statistics struct {
		Reads []struct {
			Count         int64 `xml:"count,attr"`
			AverageLength int   `xml:"average,attr"`
		} `xml:"Read"`
	} `xml:"Statistics"`
}
