# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /processing ./cmd/processing

# Runtime stage
FROM openjdk:17-slim

WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install Trimmomatic
RUN wget -q http://www.usadellab.org/cms/uploads/supplementary/Trimmomatic/Trimmomatic-0.39.zip \
    && unzip Trimmomatic-0.39.zip \
    && mv Trimmomatic-0.39 /opt/trimmomatic \
    && rm Trimmomatic-0.39.zip

# Copy binary from builder
COPY --from=builder /processing /app/processing

# Copy configuration
COPY configs/ /app/configs/

# Create data directories
RUN mkdir -p /data/processing /data/output /tmp/processing

# Set environment variables
ENV TRIMMOMATIC_JAR=/opt/trimmomatic/trimmomatic-0.39.jar
ENV TRIMMOMATIC_ADAPTERS=/opt/trimmomatic/adapters
ENV DATA_DIR=/data/processing
ENV TEMP_DIR=/tmp/processing
ENV OUTPUT_DIR=/data/output

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["/app/processing", "-config", "/app/configs/config.yaml"]
