# Build stage for Go
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /analysis ./cmd/analysis

# Runtime stage with R
FROM r-base:4.3.2

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    libcurl4-openssl-dev \
    libssl-dev \
    libxml2-dev \
    libfontconfig1-dev \
    libharfbuzz-dev \
    libfribidi-dev \
    libfreetype6-dev \
    libpng-dev \
    libtiff5-dev \
    libjpeg-dev \
    && rm -rf /var/lib/apt/lists/*

# Install R packages
RUN R -e "install.packages(c('jsonlite', 'tidyverse', 'ggplot2', 'pheatmap', 'RColorBrewer'), repos='https://cran.r-project.org')"

# Install Bioconductor packages
RUN R -e "if (!require('BiocManager', quietly = TRUE)) install.packages('BiocManager', repos='https://cran.r-project.org'); BiocManager::install(c('DESeq2', 'edgeR', 'limma'), ask=FALSE)"

# Install Kallisto
RUN wget -q https://github.com/pachterlab/kallisto/releases/download/v0.48.0/kallisto_linux-v0.48.0.tar.gz \
    && tar -xzf kallisto_linux-v0.48.0.tar.gz \
    && mv kallisto /opt/kallisto \
    && rm kallisto_linux-v0.48.0.tar.gz

# Copy Go binary
COPY --from=go-builder /analysis /app/analysis

# Copy R scripts
COPY r_scripts/ /app/r_scripts/

# Copy configuration
COPY configs/ /app/configs/

# Create directories
RUN mkdir -p /data/analysis /data/results /tmp/analysis

# Set environment
ENV KALLISTO_PATH=/opt/kallisto/kallisto
ENV R_PATH=/usr/bin/Rscript

# Expose port
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# Run
CMD ["/app/analysis", "-config", "/app/configs/config.yaml"]
