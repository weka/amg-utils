#!/bin/bash

set -e

OCI_REGISTRY="oci://ghcr.io/sdimitro"
CHARTS_DIR="/home/serapheim/src/amg-utils/charts"
GITHUB_USER="sdimitro"

echo "🚀 Publishing AMG Helm charts to OCI registry..."

# Check if GITHUB_TOKEN is set
if [ -z "${GITHUB_TOKEN}" ]; then
    echo "❌ Error: GITHUB_TOKEN environment variable is not set"
    echo "💡 Please set your GitHub token:"
    echo "   export GITHUB_TOKEN=your_github_token_here"
    echo "   Or run: echo 'your_token' | helm registry login ghcr.io -u ${GITHUB_USER} --password-stdin"
    exit 1
fi

# Login to GitHub Container Registry
echo "🔐 Logging into GitHub Container Registry..."
echo "${GITHUB_TOKEN}" | helm registry login ghcr.io -u "${GITHUB_USER}" --password-stdin
if [ $? -eq 0 ]; then
    echo "  ✅ Successfully logged into GHCR"
else
    echo "  ❌ Failed to login to GHCR"
    exit 1
fi

# Ensure we're in the charts directory
cd "${CHARTS_DIR}"

# Generate packages
echo "📦 Generating chart packages..."
./package-generate.sh

# Get chart versions dynamically
AMG_CHART_VERSION=$(grep '^version:' amg-chart/Chart.yaml | awk '{print $2}')
AMG_CW_CHART_VERSION=$(grep '^version:' amg-cw-chart/Chart.yaml | awk '{print $2}')

echo "📋 Detected chart versions:"
echo "  amg-chart: ${AMG_CHART_VERSION}"
echo "  amg-cw-chart: ${AMG_CW_CHART_VERSION}"

# Push charts to OCI registry
echo "⬆️  Pushing charts to ${OCI_REGISTRY}..."

# Push amg-chart
AMG_CHART_PACKAGE="amg-chart-${AMG_CHART_VERSION}.tgz"
if [ -f "${AMG_CHART_PACKAGE}" ]; then
    echo "  Pushing ${AMG_CHART_PACKAGE}..."
    helm push "${AMG_CHART_PACKAGE}" "${OCI_REGISTRY}"
    echo "  ✅ amg-chart published successfully"
else
    echo "  ❌ amg-chart package not found: ${AMG_CHART_PACKAGE}"
    exit 1
fi

# Push amg-cw-chart
AMG_CW_CHART_PACKAGE="amg-cw-chart-${AMG_CW_CHART_VERSION}.tgz"
if [ -f "${AMG_CW_CHART_PACKAGE}" ]; then
    echo "  Pushing ${AMG_CW_CHART_PACKAGE}..."
    helm push "${AMG_CW_CHART_PACKAGE}" "${OCI_REGISTRY}"
    echo "  ✅ amg-cw-chart published successfully"
else
    echo "  ❌ amg-cw-chart package not found: ${AMG_CW_CHART_PACKAGE}"
    exit 1
fi

# Clean up packages
echo "🧹 Cleaning up local packages..."
rm -f *.tgz

echo "🎉 All charts published successfully!"
echo "📋 Installation commands:"
echo "  helm install my-amg ${OCI_REGISTRY}/amg-chart --version ${AMG_CHART_VERSION}"
echo "  helm install my-amg-cw ${OCI_REGISTRY}/amg-cw-chart --version ${AMG_CW_CHART_VERSION}"
