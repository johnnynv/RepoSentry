#!/bin/bash

# RepoSentry Bootstrap Pipeline Installation Script
# This script deploys the static Bootstrap Pipeline infrastructure to a Kubernetes cluster

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEFAULT_OUTPUT_DIR="$PROJECT_ROOT/deployments/tekton/bootstrap"
DEFAULT_NAMESPACE="reposentry-system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration variables
OUTPUT_DIR="${BOOTSTRAP_OUTPUT_DIR:-$DEFAULT_OUTPUT_DIR}"
SYSTEM_NAMESPACE="${BOOTSTRAP_NAMESPACE:-$DEFAULT_NAMESPACE}"
DRY_RUN="${BOOTSTRAP_DRY_RUN:-false}"
SKIP_GENERATION="${BOOTSTRAP_SKIP_GENERATION:-false}"
VERBOSE="${BOOTSTRAP_VERBOSE:-false}"

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

print_verbose() {
    if [ "$VERBOSE" = "true" ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Install RepoSentry Bootstrap Pipeline infrastructure to Kubernetes cluster.

OPTIONS:
    -h, --help                    Show this help message
    -n, --namespace NAMESPACE     System namespace (default: $DEFAULT_NAMESPACE)
    -o, --output DIR             Output directory (default: $DEFAULT_OUTPUT_DIR)
    -d, --dry-run                Perform dry run without actual deployment
    -s, --skip-generation        Skip YAML generation (use existing files)
    -v, --verbose                Enable verbose output
    --force                      Force deployment even if resources exist

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE          Override system namespace
    BOOTSTRAP_OUTPUT_DIR         Override output directory
    BOOTSTRAP_DRY_RUN           Set to 'true' for dry run
    BOOTSTRAP_SKIP_GENERATION   Set to 'true' to skip generation
    BOOTSTRAP_VERBOSE           Set to 'true' for verbose output

EXAMPLES:
    # Basic installation
    $0

    # Install to custom namespace
    $0 --namespace my-reposentry-system

    # Dry run to see what would be deployed
    $0 --dry-run

    # Use existing generated files
    $0 --skip-generation

    # Verbose installation
    $0 --verbose

EOF
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed or not in PATH"
        print_error "Please install kubectl: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi

    # Check if we can connect to Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        print_error "Please ensure kubectl is configured and cluster is accessible"
        exit 1
    fi

    # Check if Tekton is installed
    if ! kubectl api-resources | grep -q "tekton.dev"; then
        print_warning "Tekton API resources not found in cluster"
        print_warning "Please ensure Tekton Pipelines is installed: https://tekton.dev/docs/installation/"
        print_warning "Continuing anyway - deployment may fail if Tekton is not properly installed"
    fi

    # Check if reposentry binary exists or can be built
    if [ "$SKIP_GENERATION" = "false" ]; then
        if [ ! -f "$PROJECT_ROOT/reposentry" ]; then
            print_status "RepoSentry binary not found, checking if we can build it..."
            if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
                print_error "Not in a RepoSentry project directory"
                exit 1
            fi
        fi
    fi

    print_success "Prerequisites check completed"
}

# Function to generate Bootstrap Pipeline YAML
generate_bootstrap_yaml() {
    if [ "$SKIP_GENERATION" = "true" ]; then
        print_status "Skipping YAML generation (using existing files)"
        return
    fi

    print_status "Generating Bootstrap Pipeline YAML files..."

    # Ensure we have the reposentry binary
    if [ ! -f "$PROJECT_ROOT/reposentry" ]; then
        print_status "Building RepoSentry binary..."
        (cd "$PROJECT_ROOT" && go build -o reposentry ./cmd/reposentry/)
        if [ $? -ne 0 ]; then
            print_error "Failed to build RepoSentry binary"
            exit 1
        fi
    fi

    # Generate the Bootstrap Pipeline resources
    print_verbose "Output directory: $OUTPUT_DIR"
    print_verbose "System namespace: $SYSTEM_NAMESPACE"

    "$PROJECT_ROOT/reposentry" generate bootstrap-pipeline \
        --output "$OUTPUT_DIR" \
        --system-namespace "$SYSTEM_NAMESPACE"

    if [ $? -ne 0 ]; then
        print_error "Failed to generate Bootstrap Pipeline YAML"
        exit 1
    fi

    print_success "Bootstrap Pipeline YAML generated successfully"
}

# Function to verify generated files
verify_generated_files() {
    print_status "Verifying generated files..."

    local required_files=(
        "00-namespace.yaml"
        "01-pipeline.yaml"
        "02-tasks.yaml"
        "03-serviceaccount.yaml"
        "04-role.yaml"
        "05-rolebinding.yaml"
    )

    for file in "${required_files[@]}"; do
        if [ ! -f "$OUTPUT_DIR/$file" ]; then
            print_error "Required file not found: $OUTPUT_DIR/$file"
            exit 1
        fi
        print_verbose "Found: $file"
    done

    print_success "All required files verified"
}

# Function to deploy Bootstrap Pipeline
deploy_bootstrap_pipeline() {
    print_status "Deploying Bootstrap Pipeline infrastructure..."

    if [ "$DRY_RUN" = "true" ]; then
        print_warning "DRY RUN MODE - No actual changes will be made"
        DRY_RUN_FLAG="--dry-run=client"
    else
        DRY_RUN_FLAG=""
    fi

    # Apply files in order
    local files=(
        "00-namespace.yaml"
        "01-pipeline.yaml"
        "02-tasks.yaml"
        "03-serviceaccount.yaml"
        "04-role.yaml"
        "05-rolebinding.yaml"
    )

    for file in "${files[@]}"; do
        print_status "Applying $file..."
        kubectl apply -f "$OUTPUT_DIR/$file" $DRY_RUN_FLAG
        if [ $? -ne 0 ]; then
            print_error "Failed to apply $file"
            exit 1
        fi
        print_verbose "Successfully applied $file"
    done

    if [ "$DRY_RUN" = "true" ]; then
        print_success "DRY RUN completed - Bootstrap Pipeline would be deployed successfully"
    else
        print_success "Bootstrap Pipeline deployed successfully"
    fi
}

# Function to verify deployment
verify_deployment() {
    if [ "$DRY_RUN" = "true" ]; then
        print_status "Skipping deployment verification (dry run mode)"
        return
    fi

    print_status "Verifying deployment..."

    # Check namespace
    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Namespace '$SYSTEM_NAMESPACE' exists"
    else
        print_error "Namespace '$SYSTEM_NAMESPACE' not found"
        exit 1
    fi

    # Check pipeline
    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Bootstrap Pipeline exists"
    else
        print_error "Bootstrap Pipeline not found"
        exit 1
    fi

    # Check tasks
    local expected_tasks=(
        "reposentry-bootstrap-clone"
        "reposentry-bootstrap-compute-namespace"
        "reposentry-bootstrap-validate"
        "reposentry-bootstrap-ensure-namespace"
        "reposentry-bootstrap-apply"
    )

    for task in "${expected_tasks[@]}"; do
        if kubectl get task "$task" -n "$SYSTEM_NAMESPACE" &> /dev/null; then
            print_verbose "Task '$task' exists"
        else
            print_error "Task '$task' not found"
            exit 1
        fi
    done
    print_success "All Bootstrap Tasks exist"

    # Check RBAC
    if kubectl get serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Bootstrap ServiceAccount exists"
    else
        print_error "Bootstrap ServiceAccount not found"
        exit 1
    fi

    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_success "Bootstrap ClusterRole exists"
    else
        print_error "Bootstrap ClusterRole not found"
        exit 1
    fi

    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        print_success "Bootstrap ClusterRoleBinding exists"
    else
        print_error "Bootstrap ClusterRoleBinding not found"
        exit 1
    fi

    print_success "Deployment verification completed successfully"
}

# Function to show post-installation information
show_post_install_info() {
    cat << EOF

${GREEN}🎉 Bootstrap Pipeline Installation Completed!${NC}

${BLUE}📋 Summary:${NC}
  • System Namespace: ${SYSTEM_NAMESPACE}
  • Bootstrap Pipeline: reposentry-bootstrap-pipeline
  • Bootstrap Tasks: 5 tasks deployed
  • RBAC: ServiceAccount, ClusterRole, ClusterRoleBinding configured

${BLUE}📁 Generated Files:${NC}
  • Location: ${OUTPUT_DIR}
  • README.md: Detailed deployment instructions

${BLUE}🔍 Verification Commands:${NC}
  # Check all resources
  kubectl get all -n ${SYSTEM_NAMESPACE}

  # Check RBAC
  kubectl get serviceaccount,clusterrole,clusterrolebinding | grep reposentry-bootstrap

  # View Pipeline definition
  kubectl describe pipeline reposentry-bootstrap-pipeline -n ${SYSTEM_NAMESPACE}

${BLUE}📚 Next Steps:${NC}
  1. Configure RepoSentry with Tekton enabled
  2. Start RepoSentry - it will automatically use the Bootstrap Pipeline
  3. Monitor PipelineRuns: kubectl get pipelineruns -n ${SYSTEM_NAMESPACE}

${BLUE}🔧 Troubleshooting:${NC}
  • Check logs: kubectl logs -n ${SYSTEM_NAMESPACE} -l tekton.dev/pipeline=reposentry-bootstrap-pipeline
  • See README.md in ${OUTPUT_DIR} for detailed troubleshooting

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -n|--namespace)
            SYSTEM_NAMESPACE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -s|--skip-generation)
            SKIP_GENERATION="true"
            shift
            ;;
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        --force)
            FORCE="true"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main installation flow
main() {
    echo -e "${GREEN}🚀 Installing RepoSentry Bootstrap Pipeline...${NC}"
    echo

    check_prerequisites
    generate_bootstrap_yaml
    verify_generated_files
    deploy_bootstrap_pipeline
    verify_deployment
    show_post_install_info
}

# Run main function
main "$@"

