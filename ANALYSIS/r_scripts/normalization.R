#!/usr/bin/env Rscript
# Count Normalization (TPM, RPKM, CPM)
# Usage: Rscript normalization.R args.json output.json

suppressPackageStartupMessages({
  library(jsonlite)
})

# Read arguments
args <- commandArgs(trailingOnly = TRUE)
if (length(args) < 2) {
  stop("Usage: Rscript normalization.R args.json output.json")
}

args_file <- args[1]
output_file <- args[2]

params <- fromJSON(args_file)

cat("Loading data...\n")

# Read counts matrix
counts <- read.csv(params$counts_file, row.names = 1, check.names = FALSE)

# Read gene lengths if provided
gene_lengths <- NULL
if (!is.null(params$lengths_file)) {
  gene_lengths <- read.csv(params$lengths_file, row.names = 1)
  gene_lengths <- gene_lengths[rownames(counts), 1]
}

cat(sprintf("Samples: %d, Genes: %d\n", ncol(counts), nrow(counts)))

# Calculate CPM (Counts Per Million)
calculate_cpm <- function(counts) {
  lib_sizes <- colSums(counts)
  t(t(counts) / lib_sizes * 1e6)
}

# Calculate RPKM (Reads Per Kilobase per Million)
calculate_rpkm <- function(counts, lengths) {
  if (is.null(lengths)) {
    stop("Gene lengths required for RPKM")
  }
  lib_sizes <- colSums(counts)
  rpkm <- t(t(counts) / lib_sizes * 1e6)
  rpkm <- rpkm / (lengths / 1000)
  return(rpkm)
}

# Calculate TPM (Transcripts Per Million)
calculate_tpm <- function(counts, lengths) {
  if (is.null(lengths)) {
    stop("Gene lengths required for TPM")
  }
  rate <- counts / lengths
  tpm <- t(t(rate) / colSums(rate) * 1e6)
  return(tpm)
}

# Normalize based on method
method <- ifelse(is.null(params$method), "cpm", params$method)

cat(sprintf("Normalization method: %s\n", method))

result <- switch(method,
  "cpm" = calculate_cpm(counts),
  "rpkm" = calculate_rpkm(counts, gene_lengths),
  "tpm" = calculate_tpm(counts, gene_lengths),
  stop(sprintf("Unknown method: %s", method))
)

# Prepare output
output <- list(
  normalized_counts = as.data.frame(result),
  method = method,
  summary = list(
    n_samples = ncol(result),
    n_genes = nrow(result),
    method = method
  )
)

# Save normalized counts to file
if (!is.null(params$output_counts_file)) {
  write.csv(result, params$output_counts_file)
  cat(sprintf("Saved normalized counts to: %s\n", params$output_counts_file))
}

# Write output
cat("Writing results...\n")
write_json(output, output_file, auto_unbox = TRUE, pretty = TRUE)

cat("Done!\n")
