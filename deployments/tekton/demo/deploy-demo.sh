#!/bin/bash

# RepoSentry Tekton Demo Deployment Script
# This script deploys the demo pipeline and runs it

set -e

echo "🚀 RepoSentry Tekton Demo Deployment"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if we can connect to the cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    exit 1
fi

print_status "Checking Tekton installation..."
if ! kubectl get crd pipelines.tekton.dev &> /dev/null; then
    print_error "Tekton Pipelines is not installed in this cluster"
    print_status "Please install Tekton Pipelines first:"
    echo "  kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"
    exit 1
fi

print_success "Tekton Pipelines is installed"

# Check if git-clone ClusterTask exists
print_status "Checking for git-clone ClusterTask..."
if ! kubectl get clustertask git-clone &> /dev/null; then
    print_warning "git-clone ClusterTask not found, installing..."
    kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml
    print_success "git-clone ClusterTask installed"
else
    print_success "git-clone ClusterTask already exists"
fi

# Create demo namespace
print_status "Creating demo namespace..."
kubectl create namespace reposentry-demo --dry-run=client -o yaml | kubectl apply -f -
print_success "Demo namespace ready"

# Apply the demo pipeline
print_status "Applying demo pipeline..."
kubectl apply -f "$(dirname "$0")/demo-pipeline.yaml"
print_success "Demo pipeline applied"

# Wait a moment for the pipeline to be registered
sleep 2

# Verify pipeline was created
if kubectl get pipeline reposentry-demo-pipeline -n default &> /dev/null; then
    print_success "Pipeline 'reposentry-demo-pipeline' created successfully"
else
    print_error "Failed to create pipeline"
    exit 1
fi

# Ask user if they want to run the pipeline
echo ""
read -p "$(echo -e ${BLUE}Do you want to run the demo pipeline now? [y/N]:${NC} )" -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Creating and running PipelineRun..."
    
    # Apply the PipelineRun
    PIPELINERUN_NAME=$(kubectl create -f "$(dirname "$0")/demo-pipelinerun.yaml" -o jsonpath='{.metadata.name}')
    print_success "PipelineRun '$PIPELINERUN_NAME' created"
    
    echo ""
    print_status "You can monitor the pipeline execution with:"
    echo "  kubectl logs -f pipelinerun/$PIPELINERUN_NAME -n default"
    echo ""
    print_status "Or check the status with:"
    echo "  kubectl get pipelinerun $PIPELINERUN_NAME -n default"
    echo ""
    print_status "To view in Tekton Dashboard (if installed):"
    echo "  kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 9097:9097"
    echo "  Then visit: http://localhost:9097"
    
    # Optionally follow logs
    echo ""
    read -p "$(echo -e ${BLUE}Do you want to follow the logs now? [y/N]:${NC} )" -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Following logs for PipelineRun '$PIPELINERUN_NAME'..."
        kubectl logs -f pipelinerun/$PIPELINERUN_NAME -n default
    fi
else
    print_status "Pipeline created but not executed"
    print_status "To run it manually:"
    echo "  kubectl create -f $(dirname "$0")/demo-pipelinerun.yaml"
fi

echo ""
print_success "Demo deployment completed!"
print_status "Pipeline: reposentry-demo-pipeline"
print_status "Target repository: https://github.com/johnnynv-org/tekton-workflow-gh-demo"
print_status "Demo namespace: reposentry-demo"


