# ANALYSIS Module

**Módulo de análise estatística de sequências biológicas**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![R](https://img.shields.io/badge/R-4.3+-276DC3?style=flat&logo=r)](https://www.r-project.org/)

## Descrição

O módulo **ANALYSIS** é responsável pela análise estatística de dados de sequenciamento, especialmente RNA-seq. Combina a performance do Go para orquestração de pipelines com o poder estatístico do R para análises bioinformáticas.

## Funcionalidades

### 📈 Quantificação de Expressão Gênica
- Integração com **RSEM** (RNA-Seq by Expectation-Maximization)
- Suporte a **Kallisto** e **Salmon** (pseudo-alignment)
- Cálculo de métricas: RPKM, FPKM, TPM
- Normalização de dados

### 📊 Análises Estatísticas
- Análise de expressão diferencial
- Clustering hierárquico
- Análise de componentes principais (PCA)
- Testes estatísticos (t-test, ANOVA, etc.)

### 🧬 Análises Bioinformáticas
- Anotação funcional de genes
- Enriquecimento de vias (pathway enrichment)
- Gene Ontology (GO) analysis
- KEGG pathway analysis

## Estrutura

```
ANALYSIS/
├── cmd/
│   └── analysis/
│       └── main.go              # Ponto de entrada
├── internal/
│   ├── quantify/
│   │   ├── rsem.go              # Wrapper RSEM
│   │   ├── kallisto.go          # Wrapper Kallisto
│   │   ├── salmon.go            # Wrapper Salmon
│   │   └── metrics.go           # Cálculo de métricas
│   ├── stats/
│   │   ├── differential.go      # Expressão diferencial
│   │   ├── clustering.go        # Clustering
│   │   ├── pca.go               # Análise PCA
│   │   └── tests.go             # Testes estatísticos
│   ├── pipeline/
│   │   ├── runner.go            # Executor de pipelines
│   │   └── workflow.go          # Definição de workflows
│   ├── rbridge/
│   │   ├── executor.go          # Execução de scripts R
│   │   └── parser.go            # Parser de resultados R
│   └── config/
│       └── config.go            # Configurações
├── r_scripts/
│   ├── differential_expression.R
│   ├── clustering.R
│   ├── pca_analysis.R
│   ├── normalization.R
│   ├── go_enrichment.R
│   └── kegg_pathway.R
├── pkg/
│   ├── matrix/                  # Operações com matrizes
│   └── stats/                   # Funções estatísticas
├── go.mod
├── go.sum
└── README.md
```

## Dependências Externas

### RSEM
```bash
# Instalação via conda
conda install -c bioconda rsem

# Ou compilar do fonte
git clone https://github.com/deweylab/RSEM.git
cd RSEM && make
export PATH=$PATH:$(pwd)
```

### Kallisto
```bash
# Instalação via conda
conda install -c bioconda kallisto

# Ou download binário
wget https://github.com/pachterlab/kallisto/releases/download/v0.48.0/kallisto_linux-v0.48.0.tar.gz
```

### R Packages
```r
# Pacotes necessários
install.packages(c(
  "tidyverse",
  "ggplot2",
  "pheatmap",
  "RColorBrewer"
))

# Bioconductor packages
if (!require("BiocManager", quietly = TRUE))
    install.packages("BiocManager")

BiocManager::install(c(
  "DESeq2",
  "edgeR",
  "limma",
  "clusterProfiler",
  "org.Dm.eg.db",
  "KEGGREST"
))
```

## Configuração

### Variáveis de Ambiente
```bash
# API do módulo CONTROL
CONTROL_API_URL=http://localhost:8080

# Ferramentas de quantificação
RSEM_PATH=/opt/rsem
KALLISTO_PATH=/opt/kallisto
SALMON_PATH=/opt/salmon

# R
R_HOME=/usr/lib/R
R_LIBS_USER=/home/user/R/library

# Diretórios
DATA_DIR=/data/analysis
OUTPUT_DIR=/data/results
```

### Arquivo de Configuração (config.yaml)
```yaml
quantification:
  default_tool: kallisto
  threads: 8
  
  rsem:
    path: /opt/rsem
    bowtie2_path: /opt/bowtie2
    
  kallisto:
    path: /opt/kallisto
    bootstrap: 100

r:
  timeout: 3600  # 1 hora
  memory_limit: 8G
  scripts_path: ./r_scripts

analysis:
  pvalue_threshold: 0.05
  log2fc_threshold: 1.0
```

## Pipelines de Análise

### 1. Quantificação RNA-seq
```
FASTQ files → Kallisto/RSEM → Count Matrix → Normalization → TPM/RPKM
```

### 2. Expressão Diferencial
```
Count Matrix → DESeq2/edgeR → Statistical Tests → Significant Genes
```

### 3. Análise Funcional
```
Gene List → GO Enrichment → KEGG Pathways → Functional Annotation
```

## Uso

### Inicialização
```bash
# Inicializar módulo Go
go mod init github.com/guidiju-50/pandora/ANALYSIS

# Instalar dependências
go mod tidy

# Compilar
go build -o bin/analysis cmd/analysis/main.go

# Executar
./bin/analysis
```

### Exemplo: Quantificação com Kallisto
```go
// Configurar quantificação
quant := quantify.NewKallisto(config)

// Executar quantificação
result, err := quant.Run(quantify.Input{
    Reads1:    "/data/sample_R1.fastq.gz",
    Reads2:    "/data/sample_R2.fastq.gz",
    Index:     "/data/transcriptome.idx",
    Bootstrap: 100,
    Threads:   8,
})

// Calcular métricas
metrics := result.CalculateTPM()
```

### Exemplo: Análise Estatística com R
```go
// Criar bridge para R
rbridge := rbridge.New(config)

// Executar análise de expressão diferencial
result, err := rbridge.Execute("differential_expression.R", map[string]interface{}{
    "counts_file":   "/data/counts.csv",
    "metadata_file": "/data/metadata.csv",
    "pvalue":        0.05,
    "log2fc":        1.0,
})
```

## Scripts R

### differential_expression.R
Análise de expressão diferencial usando DESeq2:
- Normalização de contagens
- Teste de Wald para significância
- Correção de múltiplos testes (Benjamini-Hochberg)
- Geração de volcano plots e MA plots

### clustering.R
Clustering hierárquico de amostras:
- Distância euclidiana
- Método de ligação (complete, average, ward)
- Heatmaps com anotações

### pca_analysis.R
Análise de componentes principais:
- Redução de dimensionalidade
- Visualização de agrupamentos
- Identificação de outliers

## API Interna

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/jobs/quantify` | Iniciar quantificação |
| POST | `/jobs/differential` | Análise diferencial |
| POST | `/jobs/enrichment` | Enriquecimento funcional |
| GET | `/jobs/{id}/status` | Status do job |
| GET | `/jobs/{id}/results` | Resultados |
| GET | `/health` | Health check |

## Métricas de Expressão

| Métrica | Descrição |
|---------|-----------|
| **RPKM** | Reads Per Kilobase of transcript per Million mapped reads |
| **FPKM** | Fragments Per Kilobase of transcript per Million mapped reads |
| **TPM** | Transcripts Per Million |
| **CPM** | Counts Per Million |

## Referências

- Li, B. & Dewey, C.N. (2011). RSEM: accurate transcript quantification from RNA-Seq data. BMC Bioinformatics.
- Bray, N.L. et al. (2016). Near-optimal probabilistic RNA-seq quantification. Nature Biotechnology.
- Wagner, G.P. et al. (2012). Measurement of mRNA abundance using RNA-seq data: RPKM measure is inconsistent among samples.
- Venables, W.N. & Smith, D.M. (2013). An Introduction to R.
