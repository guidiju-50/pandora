#!/usr/bin/env Rscript
# PCA Analysis for RNA-seq data
# Usage: Rscript pca_analysis.R args.json output.json

suppressPackageStartupMessages({
  library(jsonlite)
})

# Read arguments
args <- commandArgs(trailingOnly = TRUE)
if (length(args) < 2) {
  stop("Usage: Rscript pca_analysis.R args.json output.json")
}

args_file <- args[1]
output_file <- args[2]

params <- fromJSON(args_file)

cat("Loading data...\n")

# Read counts matrix
counts <- read.csv(params$counts_file, row.names = 1, check.names = FALSE)

# Read metadata
metadata <- read.csv(params$metadata_file, row.names = 1)

# Ensure sample order matches
common_samples <- intersect(colnames(counts), rownames(metadata))
counts <- counts[, common_samples]
metadata <- metadata[common_samples, , drop = FALSE]

cat(sprintf("Samples: %d, Genes: %d\n", ncol(counts), nrow(counts)))

# Log transform (add pseudocount)
log_counts <- log2(counts + 1)

# Filter low variance genes
vars <- apply(log_counts, 1, var)
top_genes <- names(sort(vars, decreasing = TRUE))[1:min(1000, nrow(counts))]
log_counts <- log_counts[top_genes, ]

cat(sprintf("Using top %d variable genes\n", nrow(log_counts)))

# Transpose for PCA (samples as rows)
t_counts <- t(log_counts)

# Run PCA
cat("Running PCA...\n")
n_components <- min(params$n_components, ncol(t_counts), nrow(t_counts) - 1)
pca_result <- prcomp(t_counts, center = TRUE, scale. = TRUE)

# Calculate variance explained
var_explained <- (pca_result$sdev^2) / sum(pca_result$sdev^2)

# Get sample scores
sample_scores <- as.data.frame(pca_result$x[, 1:n_components])
sample_scores$sample <- rownames(sample_scores)

# Add condition from metadata
if ("condition" %in% colnames(metadata)) {
  sample_scores$condition <- metadata$condition[match(sample_scores$sample, rownames(metadata))]
} else {
  sample_scores$condition <- "unknown"
}

cat(sprintf("PC1: %.1f%%, PC2: %.1f%%\n", 
            var_explained[1] * 100, var_explained[2] * 100))

# Prepare output
output <- list(
  variance_explained = var_explained[1:n_components],
  cumulative_variance = cumsum(var_explained)[1:n_components],
  sample_scores = lapply(1:nrow(sample_scores), function(i) {
    list(
      sample = sample_scores$sample[i],
      condition = sample_scores$condition[i],
      scores = as.numeric(sample_scores[i, 1:n_components])
    )
  }),
  loadings = list(
    genes = rownames(pca_result$rotation)[1:min(100, nrow(pca_result$rotation))],
    PC1 = pca_result$rotation[1:min(100, nrow(pca_result$rotation)), 1],
    PC2 = pca_result$rotation[1:min(100, nrow(pca_result$rotation)), 2]
  ),
  summary = list(
    n_samples = nrow(sample_scores),
    n_genes_used = nrow(log_counts),
    n_components = n_components
  )
)

# Write output
cat("Writing results...\n")
write_json(output, output_file, auto_unbox = TRUE, pretty = TRUE)

cat("Done!\n")
