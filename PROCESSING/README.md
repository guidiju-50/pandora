# PROCESSING Module

**Backend de processamento de sequências biológicas**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)

## Descrição

O módulo **PROCESSING** é responsável pela coleta, extração e preparação de dados de sequências biológicas. Implementa pipelines de ETL (Extract, Transform, Load) para automatizar o fluxo de dados desde bancos de dados públicos até o Data Warehouse do sistema.

## Funcionalidades

### 🌐 Web Scraping
- Coleta automatizada de dados do NCBI (SRA, GenBank)
- Extração de metadados de experimentos
- Download de arquivos FASTQ/FASTA
- Parsing de arquivos de anotação

### 🔄 Pipeline ETL
- **Extract**: Coleta de dados brutos de múltiplas fontes
- **Transform**: Limpeza, validação e padronização
- **Load**: Carregamento no Data Warehouse via API do CONTROL

### ✂️ Processamento de Sequências
- Integração com **Trimmomatic** para:
  - Remoção de adaptadores Illumina
  - Trimming por qualidade (LEADING, TRAILING, SLIDINGWINDOW)
  - Filtro por tamanho mínimo (MINLEN)
- Controle de qualidade pré e pós-processamento
- Geração de relatórios de qualidade

## Estrutura

```
PROCESSING/
├── cmd/
│   └── processing/
│       └── main.go           # Ponto de entrada
├── internal/
│   ├── scraper/
│   │   ├── ncbi.go           # Scraper do NCBI
│   │   ├── sra.go            # Scraper do SRA
│   │   └── parser.go         # Parsers de formatos
│   ├── etl/
│   │   ├── extract.go        # Extração de dados
│   │   ├── transform.go      # Transformação
│   │   └── load.go           # Carregamento
│   ├── trimming/
│   │   ├── trimmomatic.go    # Wrapper Trimmomatic
│   │   └── quality.go        # Controle de qualidade
│   └── config/
│       └── config.go         # Configurações
├── pkg/
│   ├── fasta/                # Utilitários FASTA
│   ├── fastq/                # Utilitários FASTQ
│   └── http/                 # Cliente HTTP
├── configs/
│   └── trimmomatic.yaml      # Configuração Trimmomatic
├── go.mod
├── go.sum
└── README.md
```

## Dependências Externas

### Trimmomatic
```bash
# Download do Trimmomatic
wget http://www.usadellab.org/cms/uploads/supplementary/Trimmomatic/Trimmomatic-0.39.zip
unzip Trimmomatic-0.39.zip

# Configurar variável de ambiente
export TRIMMOMATIC_JAR=/path/to/trimmomatic-0.39.jar
```

## Configuração

### Variáveis de Ambiente
```bash
# API do módulo CONTROL
CONTROL_API_URL=http://localhost:8080

# Trimmomatic
TRIMMOMATIC_JAR=/opt/trimmomatic/trimmomatic.jar
TRIMMOMATIC_ADAPTERS=/opt/trimmomatic/adapters/

# Diretórios
DATA_DIR=/data/processing
TEMP_DIR=/tmp/processing
```

### Arquivo de Configuração (config.yaml)
```yaml
scraper:
  ncbi:
    api_key: "your_ncbi_api_key"
    rate_limit: 3  # requisições por segundo
  
trimmomatic:
  threads: 4
  leading: 3
  trailing: 3
  sliding_window: "4:15"
  min_len: 36

etl:
  batch_size: 1000
  retry_attempts: 3
```

## Uso

### Inicialização
```bash
# Inicializar módulo Go
go mod init github.com/guidiju-50/pandora/PROCESSING

# Instalar dependências
go mod tidy

# Compilar
go build -o bin/processing cmd/processing/main.go

# Executar
./bin/processing
```

### Exemplo de Pipeline
```go
// Criar pipeline ETL
pipeline := etl.NewPipeline(config)

// Extrair dados do NCBI
data, err := pipeline.Extract("SRR12345678")

// Processar com Trimmomatic
cleaned, err := pipeline.Transform(data, trimmomatic.Options{
    Leading:  3,
    Trailing: 3,
    MinLen:   36,
})

// Carregar no Data Warehouse
err = pipeline.Load(cleaned)
```

## API Interna

O módulo expõe endpoints internos para integração:

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/jobs/scrape` | Iniciar job de scraping |
| POST | `/jobs/process` | Processar sequências |
| GET | `/jobs/{id}/status` | Status do job |
| GET | `/health` | Health check |

## Referências

- Bolger, A.M. et al. (2014). Trimmomatic: a flexible trimmer for Illumina sequence data. Bioinformatics 30(15): 2114-2120.
- Gheorghe, M. et al. (2018). Modern techniques of web scraping for data scientists.
- Kimball, R. & Caserta, J. (2011). The Data Warehouse ETL Toolkit.
