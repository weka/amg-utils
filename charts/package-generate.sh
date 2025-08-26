#!/bin/bash

CHART_DIR="amg-chart"
DEPS_DIR="${CHART_DIR}/charts"

# Clean up old dependencies
rm -rf ${DEPS_DIR}/*.tgz ${DEPS_DIR}/*-operator

# Download dependencies
helm dependency update ${CHART_DIR}

# Extract the downloaded dependencies
for f in ${DEPS_DIR}/*.tgz; do
  tar -xzvf "$f" -C ${DEPS_DIR}
done

# Package the chart
helm package ${CHART_DIR}
