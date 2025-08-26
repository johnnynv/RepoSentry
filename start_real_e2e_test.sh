#!/bin/bash

# RepoSentry Real Tekton End-to-End Test Script
# This script tests RepoSentry with the real Tekton cluster

set -e

echo "🚀 Starting RepoSentry Real Tekton End-to-End Test"
echo "=================================================="
echo "📍 Tekton EventListener: http://webhook.10.78.14.61.nip.io/"
echo "🔗 Testing Repository: https://github.com/johnnynv/RepoSentry"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    if [ ! -z "$REPOSENTRY_PID" ]; then
        echo "Stopping RepoSentry (PID: $REPOSENTRY_PID)..."
        kill $REPOSENTRY_PID 2>/dev/null || true
    fi
    echo "Cleanup completed."
}

# Setup signal handlers
trap cleanup EXIT
trap cleanup INT
trap cleanup TERM

# Create logs directory
mkdir -p logs

# Step 1: Validate configuration
echo "🔧 Validating configuration..."
if ! ./reposentry config validate --config ./config.yaml; then
    echo "❌ Configuration validation failed"
    exit 1
fi
echo "✅ Configuration is valid"

# Step 2: Test webhook connectivity
echo ""
echo "📡 Testing webhook connectivity..."
if ! ./reposentry test-webhook --tekton-url http://webhook.10.78.14.61.nip.io/ --repo johnnynv/RepoSentry --branch dev; then
    echo "❌ Webhook test failed"
    exit 1
fi
echo "✅ Webhook connectivity confirmed"

# Step 3: Start RepoSentry
echo ""
echo "🔍 Starting RepoSentry monitoring..."
./reposentry run --log-level debug &
REPOSENTRY_PID=$!
echo "RepoSentry started with PID: $REPOSENTRY_PID"

# Wait for RepoSentry to start
sleep 5

# Test if RepoSentry is responding
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ RepoSentry is healthy"
else
    echo "❌ RepoSentry failed to start"
    exit 1
fi

# Step 4: Show monitoring information
echo ""
echo "📊 System Status:"
echo "- RepoSentry Health: http://localhost:8080/health"
echo "- RepoSentry Metrics: http://localhost:8080/metrics"
echo "- Tekton EventListener: http://webhook.10.78.14.61.nip.io/"
echo ""
echo "📝 Log Files:"
echo "- RepoSentry: ./logs/reposentry.log"
echo ""
echo "🎯 Test Progress:"
echo "1. ✅ Configuration validated"
echo "2. ✅ Webhook connectivity tested"
echo "3. ✅ RepoSentry monitoring started"
echo "4. 🔄 Waiting for repository polling and Tekton detection..."
echo ""
echo "📋 What RepoSentry is doing:"
echo "- Polling repository: https://github.com/johnnynv/RepoSentry"
echo "- Checking branches: main, dev"
echo "- Looking for .tekton/ directory changes"
echo "- Sending webhooks to real Tekton when changes detected"
echo ""
echo "📊 Current Tekton PipelineRuns:"
kubectl get pipelinerun -A --sort-by=.metadata.creationTimestamp | tail -3
echo ""
echo "💡 To monitor in real-time:"
echo "   Terminal 1: tail -f logs/reposentry.log"
echo "   Terminal 2: watch 'kubectl get pipelinerun -A --sort-by=.metadata.creationTimestamp | tail -5'"
echo ""
echo "🎯 Expected behavior:"
echo "- RepoSentry will poll every 1 minute"
echo "- When .tekton/ directory changes are detected, a webhook will be sent"
echo "- New PipelineRuns should appear in the Tekton cluster"
echo "- Tekton will execute the Bootstrap Pipeline workflow"
echo ""

# Step 5: Monitor for a few cycles
echo "⏰ Monitoring for 5 minutes to observe polling cycles..."
echo "Press Ctrl+C to stop early"

for i in {1..5}; do
    echo ""
    echo "📊 Monitoring cycle $i/5 (waiting 1 minute)..."
    sleep 60
    
    # Check if RepoSentry is still running
    if ! kill -0 $REPOSENTRY_PID 2>/dev/null; then
        echo "❌ RepoSentry process died unexpectedly"
        cat logs/reposentry.log | tail -20
        break
    fi
    
    # Show latest PipelineRuns
    echo "🔍 Latest PipelineRuns:"
    kubectl get pipelinerun -A --sort-by=.metadata.creationTimestamp | tail -3
    
    # Show RepoSentry health
    echo "❤️  RepoSentry Health:"
    curl -s http://localhost:8080/health | jq -r '.status // "unknown"' || echo "Could not fetch health"
done

echo ""
echo "🎉 E2E Test monitoring completed!"
echo ""
echo "📈 Final Status Summary:"
echo "- RepoSentry PID: $REPOSENTRY_PID"
echo "- Log file: logs/reposentry.log"
echo "- Health endpoint: http://localhost:8080/health"
echo ""
echo "🔍 To continue monitoring, keep this script running or check logs manually:"
echo "   tail -f logs/reposentry.log"
