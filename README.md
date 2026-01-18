# Pandora

**Automação de processos para mineração e análise de sequências biológicas**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.x-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![R](https://img.shields.io/badge/R-4.3+-276DC3?style=flat&logo=r)](https://www.r-project.org/)

## Sobre o Projeto

O **Pandora** é um sistema automatizado de mineração de dados genômicos que integra ferramentas avançadas de engenharia de software para otimizar o fluxo de trabalho de análise de sequências biológicas.

O projeto foi desenvolvido como parte do Trabalho de Conclusão de Curso (TCC) do MBA em Engenharia de Software da USP, com foco em:

- **Mineração de dados** de bancos de dados biológicos (NCBI, UniProt, etc.)
- **Processamento automatizado** de sequências de DNA/RNA
- **Análise estatística** e quantificação de expressão gênica
- **Visualização de dados** através de dashboards interativos

### Contexto de Aplicação

O sistema foi projetado para apoiar pesquisas em genômica computacional, com aplicação inicial em estudos de transcriptômica (RNA-seq) de *Helicoverpa armigera*, uma das principais pragas agrícolas do Brasil.

## Arquitetura

O sistema segue uma arquitetura modular baseada em microsserviços, com comunicação via REST APIs e filas de mensagens.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              OPERATION (Frontend)                            │
│                                   Vue.js PWA                                 │
└─────────────────────────────────────┬───────────────────────────────────────┘
                                      │ REST API
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CONTROL (Backend)                               │
│                         Go + PostgreSQL + RabbitMQ                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Auth (JWT)  │  │  REST API   │  │ Data Warehouse│ │  Message Queue     │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└───────────┬─────────────────────────────────┬───────────────────────────────┘
            │                                 │
            ▼                                 ▼
┌───────────────────────────┐     ┌───────────────────────────┐
│      PROCESSING           │     │        ANALYSIS           │
│         (Go)              │     │        (Go + R)           │
│  ┌─────────────────────┐  │     │  ┌─────────────────────┐  │
│  │ Web Scraping (ETL)  │  │     │  │ Quantificação       │  │
│  │ Trimmomatic         │  │     │  │ RSEM / Kallisto     │  │
│  │ Preparação de dados │  │     │  │ Análise Estatística │  │
│  └─────────────────────┘  │     │  └─────────────────────┘  │
└───────────────────────────┘     └───────────────────────────┘
```

## Módulos

### 📦 PROCESSING
**Backend de processamento de sequências biológicas**

- Web scraping de bancos de dados biológicos
- Pipeline ETL (Extract, Transform, Load)
- Integração com Trimmomatic para limpeza de sequências Illumina
- Preparação de dados para análise downstream

**Tecnologias:** Go

### 🎛️ CONTROL
**Backend de controle e gerenciamento**

- API REST para comunicação entre módulos
- Data Warehouse para armazenamento estruturado
- Sistema de filas com RabbitMQ
- Autenticação e autorização com JWT
- Gerenciamento de jobs e workflows

**Tecnologias:** Go, PostgreSQL, RabbitMQ

### 📊 ANALYSIS
**Módulo de análise estatística**

- Quantificação de expressão gênica (RNA-seq)
- Integração com RSEM e Kallisto/Salmon
- Análises estatísticas com R
- Cálculos de RPKM, TPM e normalização

**Tecnologias:** Go, R

### 🖥️ OPERATION
**Interface de usuário**

- Dashboard interativo para visualização de dados
- Progressive Web App (PWA)
- Gerenciamento de projetos e experimentos
- Visualização de resultados de análises

**Tecnologias:** Vue.js

## Fluxo de Trabalho

```
1. COLETA          2. PROCESSAMENTO      3. ANÁLISE           4. VISUALIZAÇÃO
   ┌─────┐            ┌─────────┐          ┌──────────┐         ┌─────────┐
   │NCBI │ ──────────▶│Trimmo-  │─────────▶│ RSEM /   │────────▶│Dashboard│
   │SRA  │  scraping  │matic    │  clean   │ Kallisto │ results │ Vue.js  │
   └─────┘            └─────────┘  reads   └──────────┘         └─────────┘
```

## Estrutura do Projeto

```
pandora/
├── PROCESSING/        # Backend de processamento (Go)
│   ├── cmd/           # Pontos de entrada
│   ├── internal/      # Lógica interna
│   │   ├── scraper/   # Web scraping
│   │   ├── etl/       # Pipeline ETL
│   │   └── trimming/  # Integração Trimmomatic
│   └── pkg/           # Pacotes reutilizáveis
│
├── CONTROL/           # Backend de controle (Go + PostgreSQL)
│   ├── cmd/           # Pontos de entrada
│   ├── internal/
│   │   ├── api/       # REST API
│   │   ├── auth/      # Autenticação JWT
│   │   ├── queue/     # RabbitMQ
│   │   └── warehouse/ # Data Warehouse
│   ├── migrations/    # Migrações do banco
│   └── pkg/
│
├── ANALYSIS/          # Análise estatística (Go + R)
│   ├── cmd/
│   ├── internal/
│   │   ├── quantify/  # Quantificação
│   │   └── stats/     # Estatísticas
│   ├── r_scripts/     # Scripts R
│   └── pkg/
│
├── OPERATION/         # Frontend (Vue.js)
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   ├── router/
│   │   ├── store/
│   │   └── assets/
│   └── public/
│
├── docker/            # Configurações Docker
├── docs/              # Documentação adicional
└── README.md
```

## Requisitos

### Sistema
- Go 1.21+
- PostgreSQL 15+
- R 4.3+
- Node.js 20+
- Docker & Docker Compose

### Ferramentas de Bioinformática
- [Trimmomatic](http://www.usadellab.org/cms/?page=trimmomatic) - Limpeza de sequências Illumina
- [RSEM](https://github.com/deweylab/RSEM) - Quantificação de transcritos
- [Kallisto](https://pachterlab.github.io/kallisto/) - Quantificação pseudo-alignment

## Instalação

```bash
# Clonar repositório
git clone https://github.com/guidiju-50/pandora.git
cd pandora

# Iniciar serviços com Docker
docker-compose up -d

# Ou iniciar cada módulo individualmente
# Ver README de cada módulo para instruções específicas
```

## Referências

Este projeto foi desenvolvido como parte do TCC do MBA em Engenharia de Software - USP (2026).

### Principais Referências Técnicas
- Cox, R. (2012). Go at Google: Language Design in the Service of Software Engineering
- Douglas, K. & Douglas, S. (2021). PostgreSQL: Up and Running. 4ed. O'Reilly Media
- You, E. (2014). Vue.js Documentation
- Bolger, A.M. et al. (2014). Trimmomatic: a flexible trimmer for Illumina sequence data

## Licença

Este projeto está em desenvolvimento como trabalho acadêmico.

## Autor

**Adriano Guilherme Silva Rocha**  
MBA Engenharia de Software - USP  
📧 guidiju@usp.br
