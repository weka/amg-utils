#!/bin/bash

# Chart directories
AMG_CHART_DIR="amg-chart"
AMG_CW_CHART_DIR="amg-cw-chart"
DEPS_DIR="${AMG_CHART_DIR}/charts"

echo "📦 Packaging Helm charts..."

# ===== AMG Chart (with dependencies) =====
echo "🔧 Processing ${AMG_CHART_DIR}..."

# Clean up old dependencies
echo "  Cleaning up old dependencies..."
rm -rf ${DEPS_DIR}/*.tgz ${DEPS_DIR}/*-operator

# Download dependencies
echo "  Updating dependencies..."
helm dependency update ${AMG_CHART_DIR}

# Extract the downloaded dependencies
echo "  Extracting dependencies..."
for f in ${DEPS_DIR}/*.tgz; do
  if [ -f "$f" ]; then
    tar -xzvf "$f" -C ${DEPS_DIR}
  fi
done

# Package the chart
echo "  Packaging ${AMG_CHART_DIR}..."
helm package ${AMG_CHART_DIR}

# ===== AMG CW Chart (no dependencies) =====
echo "🔧 Processing ${AMG_CW_CHART_DIR}..."
echo "  Packaging ${AMG_CW_CHART_DIR}..."
helm package ${AMG_CW_CHART_DIR}

echo "✅ Chart packaging complete!"
echo "Generated packages:"
ls -la *.tgz
