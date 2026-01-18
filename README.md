# Pandora

Automação de processos para mineração e análise de sequências biológicas.

## Arquitetura do Projeto

O projeto está dividido em quatro módulos principais:

| Módulo | Descrição | Tecnologias |
|--------|-----------|-------------|
| **PROCESSING** | Backend de processamento de sequências | Go |
| **CONTROL** | Backend de controle e gerenciamento | Go + PostgreSQL |
| **ANALYSIS** | Análise estatística de dados | Go + R |
| **OPERATION** | Interface de usuário | Vue.js |

## Estrutura de Pastas

```
pandora/
├── PROCESSING/    # Backend de processamento (Go)
├── CONTROL/       # Backend de controle (Go + PostgreSQL)
├── ANALYSIS/      # Módulo de análise (Go + R)
├── OPERATION/     # Frontend (Vue.js)
└── README.md
```

## Requisitos

- Go 1.21+
- PostgreSQL 15+
- R 4.3+
- Node.js 20+ (para Vue.js)

## Licença

Este projeto está em desenvolvimento.
