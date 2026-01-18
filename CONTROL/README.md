# CONTROL Module

**Backend de controle e gerenciamento de dados**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3.12+-FF6600?style=flat&logo=rabbitmq)](https://www.rabbitmq.com/)

## Descrição

O módulo **CONTROL** é o núcleo central do sistema Pandora. Responsável pelo gerenciamento de dados, orquestração de jobs, autenticação e comunicação entre os demais módulos através de REST APIs e filas de mensagens.

## Funcionalidades

### 🔐 Autenticação e Autorização
- Autenticação baseada em **JWT (JSON Web Token)**
- Controle de acesso por roles (admin, researcher, viewer)
- Sessões seguras com refresh tokens
- Proteção de endpoints sensíveis

### 🗄️ Data Warehouse
- Modelagem dimensional para dados genômicos
- Esquemas otimizados para consultas analíticas
- Versionamento de dados e auditoria
- Suporte a queries OLAP

### 📨 Sistema de Filas (RabbitMQ)
- Orquestração de jobs de processamento
- Comunicação assíncrona entre módulos
- Retry automático e dead-letter queues
- Monitoramento de filas

### 🌐 REST API
- API RESTful para frontend (OPERATION)
- Endpoints para gerenciamento de projetos
- CRUD de experimentos e amostras
- Consulta de resultados de análises

## Estrutura

```
CONTROL/
├── cmd/
│   └── control/
│       └── main.go              # Ponto de entrada
├── internal/
│   ├── api/
│   │   ├── router.go            # Configuração de rotas
│   │   ├── middleware.go        # Middlewares
│   │   └── handlers/
│   │       ├── auth.go          # Handlers de autenticação
│   │       ├── projects.go      # Handlers de projetos
│   │       ├── experiments.go   # Handlers de experimentos
│   │       ├── samples.go       # Handlers de amostras
│   │       └── jobs.go          # Handlers de jobs
│   ├── auth/
│   │   ├── jwt.go               # Geração/validação JWT
│   │   ├── password.go          # Hash de senhas
│   │   └── rbac.go              # Controle de acesso
│   ├── queue/
│   │   ├── rabbitmq.go          # Cliente RabbitMQ
│   │   ├── producer.go          # Produtor de mensagens
│   │   └── consumer.go          # Consumidor de mensagens
│   ├── warehouse/
│   │   ├── repository.go        # Repositórios
│   │   ├── models.go            # Modelos de dados
│   │   └── queries.go           # Queries SQL
│   ├── services/
│   │   ├── project.go           # Serviço de projetos
│   │   ├── experiment.go        # Serviço de experimentos
│   │   └── job.go               # Serviço de jobs
│   └── config/
│       └── config.go            # Configurações
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_projects.sql
│   ├── 003_create_experiments.sql
│   ├── 004_create_samples.sql
│   └── 005_create_jobs.sql
├── pkg/
│   ├── database/                # Utilitários de banco
│   └── validator/               # Validação de dados
├── go.mod
├── go.sum
└── README.md
```

## Modelo de Dados

### Diagrama ER Simplificado
```
┌──────────┐     ┌─────────────┐     ┌──────────┐
│  Users   │────▶│  Projects   │────▶│Experiments│
└──────────┘     └─────────────┘     └──────────┘
                        │                   │
                        ▼                   ▼
                 ┌──────────┐        ┌──────────┐
                 │   Jobs   │        │ Samples  │
                 └──────────┘        └──────────┘
                        │                   │
                        ▼                   ▼
                 ┌──────────┐        ┌──────────┐
                 │  Logs    │        │ Results  │
                 └──────────┘        └──────────┘
```

## Configuração

### Variáveis de Ambiente
```bash
# Servidor
PORT=8080
ENV=development

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_NAME=pandora
DB_USER=pandora
DB_PASSWORD=secret

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# JWT
JWT_SECRET=your-super-secret-key
JWT_EXPIRY=24h
JWT_REFRESH_EXPIRY=168h

# CORS
ALLOWED_ORIGINS=http://localhost:3000
```

### Arquivo de Configuração (config.yaml)
```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost
  port: 5432
  name: pandora
  pool_size: 25

rabbitmq:
  url: amqp://guest:guest@localhost:5672/
  queues:
    processing: pandora.processing
    analysis: pandora.analysis

jwt:
  secret: ${JWT_SECRET}
  expiry: 24h
```

## Setup do Banco de Dados

```bash
# Criar banco de dados
createdb pandora

# Executar migrações
go run cmd/migrate/main.go up

# Ou usando migrate CLI
migrate -path migrations -database "postgres://user:pass@localhost/pandora?sslmode=disable" up
```

## API Endpoints

### Autenticação
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/register` | Registro |
| POST | `/api/v1/auth/refresh` | Refresh token |
| POST | `/api/v1/auth/logout` | Logout |

### Projetos
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/api/v1/projects` | Listar projetos |
| POST | `/api/v1/projects` | Criar projeto |
| GET | `/api/v1/projects/{id}` | Detalhes do projeto |
| PUT | `/api/v1/projects/{id}` | Atualizar projeto |
| DELETE | `/api/v1/projects/{id}` | Remover projeto |

### Experimentos
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/api/v1/experiments` | Listar experimentos |
| POST | `/api/v1/experiments` | Criar experimento |
| GET | `/api/v1/experiments/{id}` | Detalhes |
| GET | `/api/v1/experiments/{id}/samples` | Amostras |
| GET | `/api/v1/experiments/{id}/results` | Resultados |

### Jobs
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/v1/jobs` | Criar job |
| GET | `/api/v1/jobs/{id}` | Status do job |
| POST | `/api/v1/jobs/{id}/cancel` | Cancelar job |

## Uso

```bash
# Inicializar módulo Go
go mod init github.com/guidiju-50/pandora/CONTROL

# Instalar dependências
go mod tidy

# Compilar
go build -o bin/control cmd/control/main.go

# Executar
./bin/control
```

## Filas RabbitMQ

| Fila | Descrição |
|------|-----------|
| `pandora.processing` | Jobs de processamento de sequências |
| `pandora.analysis` | Jobs de análise estatística |
| `pandora.notifications` | Notificações para usuários |
| `pandora.dlq` | Dead letter queue |

## Referências

- Douglas, K. & Douglas, S. (2021). PostgreSQL: Up and Running. 4ed. O'Reilly Media.
- Jones, M. et al. (2015). JSON Web Token (JWT). RFC 7519.
- Videla, A. & Williams, J. (2012). RabbitMQ Cookbook. Packt Publishing.
- Fielding, R.T. (2000). Architectural Styles and the Design of Network-based Software Architectures.
