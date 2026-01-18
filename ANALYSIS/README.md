# ANALYSIS Module

## Descrição
Módulo responsável pela análise de sequências biológicas, combinando processamento em Go com análises estatísticas em R.

## Tecnologias
- **Linguagem Principal:** Go (Golang)
- **Análise Estatística:** R

## Estrutura
```
ANALYSIS/
├── cmd/           # Pontos de entrada da aplicação Go
├── internal/      # Código interno do módulo Go
├── pkg/           # Pacotes reutilizáveis Go
├── r_scripts/     # Scripts de análise em R
└── README.md
```

## Setup
```bash
# Inicializar módulo Go
go mod init github.com/guidiju-50/pandora/ANALYSIS

# Instalar dependências R necessárias
# Rscript -e "install.packages(c('...'), repos='https://cran.r-project.org')"
```
