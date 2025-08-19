#!/bin/bash
# RepoSentry Health Check Script
# Usage: ./health_check.sh [host] [port]

HOST=${1:-localhost}
PORT=${2:-8080}
URL="http://${HOST}:${PORT}/health"

echo "🔍 Checking RepoSentry health at ${URL}"

# Perform health check
response=$(curl -s --max-time 10 "${URL}" 2>/dev/null)
curl_exit_code=$?

if [ $curl_exit_code -ne 0 ]; then
    echo "❌ Failed to connect to RepoSentry"
    echo "   URL: ${URL}"
    echo "   Error: Connection failed (exit code: $curl_exit_code)"
    exit 1
fi

# Parse JSON response
if command -v jq >/dev/null 2>&1; then
    healthy=$(echo "$response" | jq -r '.data.healthy // false' 2>/dev/null)
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null)
    
    if [ "$success" = "true" ] && [ "$healthy" = "true" ]; then
        echo "✅ RepoSentry is healthy"
        
        # Show component status
        echo "📊 Component Status:"
        echo "$response" | jq -r '.data.components | to_entries[] | "   \(.key): \(.value.status)"' 2>/dev/null || echo "   Unable to parse components"
        
        exit 0
    else
        echo "❌ RepoSentry is unhealthy"
        echo "   Response: $response"
        exit 1
    fi
else
    # Fallback without jq
    if echo "$response" | grep -q '"healthy":true'; then
        echo "✅ RepoSentry is healthy"
        exit 0
    else
        echo "❌ RepoSentry is unhealthy"
        echo "   Response: $response"
        exit 1
    fi
fi
