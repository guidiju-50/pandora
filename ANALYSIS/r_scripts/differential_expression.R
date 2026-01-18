#!/usr/bin/env Rscript
# Differential Expression Analysis using DESeq2
# Usage: Rscript differential_expression.R args.json output.json

suppressPackageStartupMessages({
  library(DESeq2)
  library(jsonlite)
})

# Read command line arguments
args <- commandArgs(trailingOnly = TRUE)
if (length(args) < 2) {
  stop("Usage: Rscript differential_expression.R args.json output.json")
}

args_file <- args[1]
output_file <- args[2]

# Load arguments
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

# Filter low count genes
if (!is.null(params$min_count)) {
  keep <- rowSums(counts >= params$min_count) >= 2
  counts <- counts[keep, ]
  cat(sprintf("After filtering: %d genes\n", nrow(counts)))
}

# Create DESeq2 dataset
# Assume metadata has a 'condition' column
dds <- DESeqDataSetFromMatrix(
  countData = round(counts),
  colData = metadata,
  design = ~ condition
)

# Set reference level
if (!is.null(params$condition2)) {
  dds$condition <- relevel(dds$condition, ref = params$condition2)
}

cat("Running DESeq2...\n")

# Run DESeq2
dds <- DESeq(dds)

# Get results
res <- results(dds, 
               contrast = c("condition", params$condition1, params$condition2),
               alpha = params$pvalue_threshold)

# Order by adjusted p-value
res <- res[order(res$padj), ]

# Convert to data frame
res_df <- as.data.frame(res)
res_df$gene_id <- rownames(res_df)

# Rename columns
colnames(res_df) <- c("baseMean", "log2FoldChange", "lfcSE", "stat", "pvalue", "padj", "gene_id")

# Add significance
res_df$significant <- !is.na(res_df$padj) & 
                       res_df$padj < params$pvalue_threshold &
                       abs(res_df$log2FoldChange) > params$log2fc_threshold

# Calculate summary
significant_genes <- res_df[res_df$significant == TRUE, ]
n_up <- sum(significant_genes$log2FoldChange > 0, na.rm = TRUE)
n_down <- sum(significant_genes$log2FoldChange < 0, na.rm = TRUE)

cat(sprintf("Significant genes: %d (up: %d, down: %d)\n", nrow(significant_genes), n_up, n_down))

# Prepare output
output <- list(
  genes = lapply(1:nrow(res_df), function(i) {
    list(
      gene_id = res_df$gene_id[i],
      gene_name = res_df$gene_id[i],  # Could be mapped to gene names
      baseMean = ifelse(is.na(res_df$baseMean[i]), 0, res_df$baseMean[i]),
      log2FoldChange = ifelse(is.na(res_df$log2FoldChange[i]), 0, res_df$log2FoldChange[i]),
      lfcSE = ifelse(is.na(res_df$lfcSE[i]), 0, res_df$lfcSE[i]),
      stat = ifelse(is.na(res_df$stat[i]), 0, res_df$stat[i]),
      pvalue = ifelse(is.na(res_df$pvalue[i]), 1, res_df$pvalue[i]),
      padj = ifelse(is.na(res_df$padj[i]), 1, res_df$padj[i])
    )
  }),
  summary = list(
    total_genes = nrow(res_df),
    significant_up = n_up,
    significant_down = n_down,
    pvalue_threshold = params$pvalue_threshold,
    log2fc_threshold = params$log2fc_threshold
  ),
  method = "DESeq2",
  comparison = paste(params$condition1, "vs", params$condition2)
)

# Write output
cat("Writing results...\n")
write_json(output, output_file, auto_unbox = TRUE, pretty = TRUE)

cat("Done!\n")
