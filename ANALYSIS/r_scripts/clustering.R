#!/usr/bin/env Rscript
# Hierarchical Clustering Analysis
# Usage: Rscript clustering.R args.json output.json

suppressPackageStartupMessages({
  library(jsonlite)
})

# Read arguments
args <- commandArgs(trailingOnly = TRUE)
if (length(args) < 2) {
  stop("Usage: Rscript clustering.R args.json output.json")
}

args_file <- args[1]
output_file <- args[2]

params <- fromJSON(args_file)

cat("Loading data...\n")

# Read counts matrix
counts <- read.csv(params$counts_file, row.names = 1, check.names = FALSE)

cat(sprintf("Samples: %d, Genes: %d\n", ncol(counts), nrow(counts)))

# Log transform
log_counts <- log2(counts + 1)

# Filter low variance genes
vars <- apply(log_counts, 1, var)
top_genes <- names(sort(vars, decreasing = TRUE))[1:min(500, nrow(counts))]
log_counts <- log_counts[top_genes, ]

cat(sprintf("Using top %d variable genes\n", nrow(log_counts)))

# Transpose for sample clustering
t_counts <- t(log_counts)

# Calculate distance matrix
method <- ifelse(is.null(params$distance), "euclidean", params$distance)
dist_matrix <- dist(t_counts, method = method)

# Perform hierarchical clustering
cluster_method <- ifelse(is.null(params$method), "ward.D2", params$method)
hc <- hclust(dist_matrix, method = cluster_method)

cat(sprintf("Clustering method: %s, Distance: %s\n", cluster_method, method))

# Cut tree to get clusters (default 3 clusters)
n_clusters <- ifelse(is.null(params$n_clusters), 3, params$n_clusters)
clusters <- cutree(hc, k = n_clusters)

# Prepare dendrogram data
dendro_data <- list(
  merge = hc$merge,
  height = hc$height,
  order = hc$order,
  labels = hc$labels
)

# Prepare cluster assignments
cluster_assignments <- lapply(1:n_clusters, function(k) {
  members <- names(clusters[clusters == k])
  list(
    id = k,
    members = members,
    size = length(members)
  )
})

# Prepare output
output <- list(
  dendrogram = dendro_data,
  clusters = cluster_assignments,
  summary = list(
    n_samples = ncol(counts),
    n_genes_used = nrow(log_counts),
    n_clusters = n_clusters,
    method = cluster_method,
    distance = method
  )
)

# Write output
cat("Writing results...\n")
write_json(output, output_file, auto_unbox = TRUE, pretty = TRUE)

cat("Done!\n")
