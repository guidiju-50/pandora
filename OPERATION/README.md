# OPERATION Module

**Interface de usuário do sistema Pandora**

[![Vue.js](https://img.shields.io/badge/Vue.js-3.x-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![Vite](https://img.shields.io/badge/Vite-5.x-646CFF?style=flat&logo=vite)](https://vitejs.dev/)
[![PWA](https://img.shields.io/badge/PWA-Ready-5A0FC8?style=flat&logo=pwa)](https://web.dev/progressive-web-apps/)

## Descrição

O módulo **OPERATION** é o frontend do sistema Pandora, desenvolvido como uma Progressive Web App (PWA) com Vue.js. Oferece uma interface moderna e responsiva para gerenciamento de projetos, visualização de dados e acompanhamento de análises.

## Funcionalidades

### 🏠 Dashboard
- Visão geral de projetos e experimentos
- Métricas e estatísticas em tempo real
- Atividades recentes
- Alertas e notificações

### 📁 Gerenciamento de Projetos
- CRUD de projetos de pesquisa
- Organização de experimentos
- Gerenciamento de amostras
- Metadados e anotações

### 📊 Visualização de Dados
- Gráficos interativos (volcano plots, heatmaps, PCA)
- Tabelas com filtros e ordenação
- Export de resultados (CSV, Excel, PDF)
- Dashboards customizáveis

### ⚙️ Gerenciamento de Jobs
- Submissão de análises
- Acompanhamento de progresso em tempo real
- Histórico de execuções
- Logs e relatórios

### 👤 Perfil e Configurações
- Gerenciamento de conta
- Preferências de usuário
- Configurações de notificação
- Temas (claro/escuro)

## Estrutura

```
OPERATION/
├── public/
│   ├── favicon.ico
│   └── manifest.json           # PWA manifest
├── src/
│   ├── assets/
│   │   ├── styles/
│   │   │   ├── main.scss       # Estilos globais
│   │   │   └── variables.scss  # Variáveis CSS
│   │   └── images/
│   ├── components/
│   │   ├── common/
│   │   │   ├── AppHeader.vue
│   │   │   ├── AppSidebar.vue
│   │   │   ├── AppFooter.vue
│   │   │   └── LoadingSpinner.vue
│   │   ├── charts/
│   │   │   ├── VolcanoPlot.vue
│   │   │   ├── Heatmap.vue
│   │   │   ├── PCAPlot.vue
│   │   │   └── BarChart.vue
│   │   ├── projects/
│   │   │   ├── ProjectCard.vue
│   │   │   ├── ProjectList.vue
│   │   │   └── ProjectForm.vue
│   │   ├── experiments/
│   │   │   ├── ExperimentCard.vue
│   │   │   ├── SampleTable.vue
│   │   │   └── ResultsViewer.vue
│   │   └── jobs/
│   │       ├── JobCard.vue
│   │       ├── JobProgress.vue
│   │       └── JobLogs.vue
│   ├── views/
│   │   ├── DashboardView.vue
│   │   ├── ProjectsView.vue
│   │   ├── ProjectDetailView.vue
│   │   ├── ExperimentsView.vue
│   │   ├── AnalysisView.vue
│   │   ├── JobsView.vue
│   │   ├── SettingsView.vue
│   │   ├── LoginView.vue
│   │   └── RegisterView.vue
│   ├── router/
│   │   └── index.js            # Configuração de rotas
│   ├── store/
│   │   ├── index.js            # Pinia store
│   │   ├── auth.js             # Estado de autenticação
│   │   ├── projects.js         # Estado de projetos
│   │   └── jobs.js             # Estado de jobs
│   ├── services/
│   │   ├── api.js              # Cliente HTTP (Axios)
│   │   ├── auth.js             # Serviço de autenticação
│   │   ├── projects.js         # API de projetos
│   │   └── jobs.js             # API de jobs
│   ├── composables/
│   │   ├── useAuth.js
│   │   ├── useNotification.js
│   │   └── useTheme.js
│   ├── utils/
│   │   ├── formatters.js
│   │   ├── validators.js
│   │   └── constants.js
│   ├── App.vue
│   └── main.js
├── .env.example
├── index.html
├── vite.config.js
├── package.json
└── README.md
```

## Tecnologias

| Categoria | Tecnologia |
|-----------|------------|
| Framework | Vue.js 3 (Composition API) |
| Build Tool | Vite |
| State Management | Pinia |
| Router | Vue Router 4 |
| HTTP Client | Axios |
| UI Components | PrimeVue / Vuetify |
| Charts | Chart.js / D3.js / Plotly |
| Styles | SCSS / Tailwind CSS |
| Testing | Vitest / Cypress |

## Configuração

### Variáveis de Ambiente
```bash
# .env.development
VITE_API_URL=http://localhost:8080/api/v1
VITE_APP_TITLE=Pandora
VITE_APP_VERSION=1.0.0

# .env.production
VITE_API_URL=https://api.pandora.example.com/api/v1
VITE_APP_TITLE=Pandora
VITE_APP_VERSION=1.0.0
```

## Instalação

```bash
# Navegar para o diretório
cd OPERATION

# Instalar dependências
npm install

# Rodar em desenvolvimento
npm run dev

# Build para produção
npm run build

# Preview do build
npm run preview

# Executar testes
npm run test

# Lint
npm run lint
```

## Rotas

| Rota | View | Descrição |
|------|------|-----------|
| `/` | DashboardView | Dashboard principal |
| `/login` | LoginView | Página de login |
| `/register` | RegisterView | Página de registro |
| `/projects` | ProjectsView | Lista de projetos |
| `/projects/:id` | ProjectDetailView | Detalhes do projeto |
| `/experiments` | ExperimentsView | Lista de experimentos |
| `/experiments/:id` | ExperimentDetailView | Detalhes do experimento |
| `/analysis` | AnalysisView | Submissão de análises |
| `/jobs` | JobsView | Lista de jobs |
| `/settings` | SettingsView | Configurações |

## Componentes de Visualização

### VolcanoPlot
Visualização de genes diferencialmente expressos:
- Eixo X: log2 Fold Change
- Eixo Y: -log10 p-value
- Destaque de genes significativos
- Tooltip com informações do gene

### Heatmap
Mapa de calor de expressão gênica:
- Clustering hierárquico
- Escala de cores customizável
- Anotações de linhas/colunas
- Zoom e pan interativos

### PCAPlot
Plot de análise de componentes principais:
- Scatter plot 2D/3D
- Agrupamento por condição
- Variância explicada
- Identificação de amostras

## PWA Features

- **Offline Support**: Service Worker para cache de assets
- **Installable**: Pode ser instalado como app nativo
- **Responsive**: Funciona em desktop, tablet e mobile
- **Push Notifications**: Alertas de jobs concluídos

## Design System

### Cores
```scss
// Cores principais
$primary: #4FC08D;      // Verde Vue
$secondary: #35495E;    // Azul escuro
$accent: #FF6B6B;       // Vermelho coral

// Semânticas
$success: #10B981;
$warning: #F59E0B;
$error: #EF4444;
$info: #3B82F6;
```

### Tipografia
```scss
$font-family-base: 'Inter', sans-serif;
$font-family-mono: 'JetBrains Mono', monospace;
```

## Referências

- You, E. (2014). Vue.js Documentation. https://vuejs.org/
- Nielsen, J. (1993). Usability Engineering. Morgan Kaufmann.
- Majchrzak, T.A. et al. (2018). Progressive web apps: The future of the mobile web?
- Freitas, C.M.S. et al. (2001). Introdução à Visualização de Informações.
