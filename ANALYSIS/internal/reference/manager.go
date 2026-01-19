// Package reference manages reference genomes and Kallisto indices.
package reference

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// OrganismInfo contains information about supported organisms.
type OrganismInfo struct {
	Name           string `json:"name"`
	ScientificName string `json:"scientific_name"`
	TaxID          string `json:"tax_id"`
	TranscriptURL  string `json:"transcript_url"`
	IndexFile      string `json:"index_file"`
	Available      bool   `json:"available"`
}

// Manager handles reference genome downloads and index management.
type Manager struct {
	referenceDir string
	kallistoPath string
	organisms    map[string]*OrganismInfo
	mu           sync.RWMutex
	logger       *zap.Logger
}

// NewManager creates a new reference manager.
func NewManager(referenceDir, kallistoPath string, logger *zap.Logger) *Manager {
	m := &Manager{
		referenceDir: referenceDir,
		kallistoPath: kallistoPath,
		organisms:    make(map[string]*OrganismInfo),
		logger:       logger,
	}

	// Register supported organisms
	m.registerOrganisms()

	return m
}

// registerOrganisms registers known organisms with their reference sources.
func (m *Manager) registerOrganisms() {
	// Helicoverpa armigera - Cotton bollworm
	m.organisms["helicoverpa_armigera"] = &OrganismInfo{
		Name:           "helicoverpa_armigera",
		ScientificName: "Helicoverpa armigera",
		TaxID:          "29058",
		TranscriptURL:  "https://ftp.ncbi.nlm.nih.gov/genomes/all/GCF/023/701/775/GCF_023701775.1_HaSCD2/GCF_023701775.1_HaSCD2_rna.fna.gz",
		IndexFile:      "helicoverpa_armigera.idx",
	}

	// Homo sapiens - Human
	m.organisms["homo_sapiens"] = &OrganismInfo{
		Name:           "homo_sapiens",
		ScientificName: "Homo sapiens",
		TaxID:          "9606",
		TranscriptURL:  "https://ftp.ensembl.org/pub/release-110/fasta/homo_sapiens/cdna/Homo_sapiens.GRCh38.cdna.all.fa.gz",
		IndexFile:      "homo_sapiens.idx",
	}

	// Mus musculus - Mouse
	m.organisms["mus_musculus"] = &OrganismInfo{
		Name:           "mus_musculus",
		ScientificName: "Mus musculus",
		TaxID:          "10090",
		TranscriptURL:  "https://ftp.ensembl.org/pub/release-110/fasta/mus_musculus/cdna/Mus_musculus.GRCm39.cdna.all.fa.gz",
		IndexFile:      "mus_musculus.idx",
	}

	// Drosophila melanogaster - Fruit fly
	m.organisms["drosophila_melanogaster"] = &OrganismInfo{
		Name:           "drosophila_melanogaster",
		ScientificName: "Drosophila melanogaster",
		TaxID:          "7227",
		TranscriptURL:  "https://ftp.ensembl.org/pub/release-110/fasta/drosophila_melanogaster/cdna/Drosophila_melanogaster.BDGP6.46.cdna.all.fa.gz",
		IndexFile:      "drosophila_melanogaster.idx",
	}

	// Arabidopsis thaliana - Thale cress
	m.organisms["arabidopsis_thaliana"] = &OrganismInfo{
		Name:           "arabidopsis_thaliana",
		ScientificName: "Arabidopsis thaliana",
		TaxID:          "3702",
		TranscriptURL:  "https://ftp.ensemblgenomes.ebi.ac.uk/pub/plants/release-57/fasta/arabidopsis_thaliana/cdna/Arabidopsis_thaliana.TAIR10.cdna.all.fa.gz",
		IndexFile:      "arabidopsis_thaliana.idx",
	}

	// Check which indices are available
	m.checkAvailability()
}

// checkAvailability checks which indices are already built.
func (m *Manager) checkAvailability() {
	for name, org := range m.organisms {
		indexPath := filepath.Join(m.referenceDir, org.IndexFile)
		if _, err := os.Stat(indexPath); err == nil {
			org.Available = true
			m.logger.Info("reference index available", zap.String("organism", name))
		}
	}
}

// GetOrganism returns organism info by name or tax ID.
func (m *Manager) GetOrganism(identifier string) (*OrganismInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try by name first
	identifier = strings.ToLower(strings.ReplaceAll(identifier, " ", "_"))
	if org, ok := m.organisms[identifier]; ok {
		return org, true
	}

	// Try by tax ID
	for _, org := range m.organisms {
		if org.TaxID == identifier {
			return org, true
		}
	}

	return nil, false
}

// ListOrganisms returns all supported organisms.
func (m *Manager) ListOrganisms() []*OrganismInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*OrganismInfo, 0, len(m.organisms))
	for _, org := range m.organisms {
		list = append(list, org)
	}
	return list
}

// GetIndexPath returns the path to the Kallisto index for an organism.
func (m *Manager) GetIndexPath(organism string) (string, error) {
	org, found := m.GetOrganism(organism)
	if !found {
		return "", fmt.Errorf("organism not found: %s", organism)
	}

	indexPath := filepath.Join(m.referenceDir, org.IndexFile)
	if !org.Available {
		return "", fmt.Errorf("index not available for %s, run EnsureIndex first", organism)
	}

	return indexPath, nil
}

// EnsureIndex ensures a Kallisto index is available, downloading and building if necessary.
func (m *Manager) EnsureIndex(ctx context.Context, organism string, progressFunc func(stage string, progress int)) error {
	org, found := m.GetOrganism(organism)
	if !found {
		return fmt.Errorf("unsupported organism: %s", organism)
	}

	indexPath := filepath.Join(m.referenceDir, org.IndexFile)

	// Check if already available
	if org.Available {
		m.logger.Info("index already available", zap.String("organism", organism))
		if progressFunc != nil {
			progressFunc("Index already available", 100)
		}
		return nil
	}

	m.logger.Info("preparing index", zap.String("organism", organism))

	// Create reference directory
	if err := os.MkdirAll(m.referenceDir, 0755); err != nil {
		return fmt.Errorf("creating reference directory: %w", err)
	}

	// Download transcriptome
	if progressFunc != nil {
		progressFunc("Downloading transcriptome", 10)
	}

	fastaPath := filepath.Join(m.referenceDir, org.Name+"_rna.fna.gz")
	if err := m.downloadFile(ctx, org.TranscriptURL, fastaPath); err != nil {
		return fmt.Errorf("downloading transcriptome: %w", err)
	}

	if progressFunc != nil {
		progressFunc("Decompressing", 50)
	}

	// Decompress
	unzippedPath := strings.TrimSuffix(fastaPath, ".gz")
	if err := m.decompressGzip(fastaPath, unzippedPath); err != nil {
		return fmt.Errorf("decompressing transcriptome: %w", err)
	}

	if progressFunc != nil {
		progressFunc("Building Kallisto index", 60)
	}

	// Build index
	if err := m.buildKallistoIndex(ctx, unzippedPath, indexPath); err != nil {
		return fmt.Errorf("building index: %w", err)
	}

	// Update availability
	m.mu.Lock()
	org.Available = true
	m.mu.Unlock()

	// Cleanup
	os.Remove(fastaPath)
	os.Remove(unzippedPath)

	if progressFunc != nil {
		progressFunc("Index ready", 100)
	}

	m.logger.Info("index built successfully", zap.String("organism", organism), zap.String("index", indexPath))
	return nil
}

// downloadFile downloads a file from URL to the specified path.
func (m *Manager) downloadFile(ctx context.Context, url, outputPath string) error {
	m.logger.Info("downloading file", zap.String("url", url), zap.String("output", outputPath))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// decompressGzip decompresses a gzip file.
func (m *Manager) decompressGzip(gzPath, outputPath string) error {
	cmd := exec.Command("gunzip", "-k", "-f", gzPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gunzip failed: %s - %w", string(output), err)
	}
	return nil
}

// buildKallistoIndex builds a Kallisto index from a FASTA file.
func (m *Manager) buildKallistoIndex(ctx context.Context, fastaPath, indexPath string) error {
	m.logger.Info("building Kallisto index", zap.String("fasta", fastaPath), zap.String("index", indexPath))

	cmd := exec.CommandContext(ctx, m.kallistoPath, "index", "-i", indexPath, fastaPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kallisto index failed: %s - %w", string(output), err)
	}

	return nil
}

// AddCustomOrganism adds a custom organism with an existing index.
func (m *Manager) AddCustomOrganism(name, scientificName, taxID, indexPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify index exists
	if _, err := os.Stat(indexPath); err != nil {
		return fmt.Errorf("index file not found: %s", indexPath)
	}

	// Copy to reference directory if not already there
	destPath := filepath.Join(m.referenceDir, filepath.Base(indexPath))
	if indexPath != destPath {
		src, err := os.Open(indexPath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	m.organisms[strings.ToLower(strings.ReplaceAll(name, " ", "_"))] = &OrganismInfo{
		Name:           name,
		ScientificName: scientificName,
		TaxID:          taxID,
		IndexFile:      filepath.Base(destPath),
		Available:      true,
	}

	return nil
}
