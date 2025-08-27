#!/bin/bash

# RepoSentry Bootstrap Pipeline Uninstall Script
# This script removes the Bootstrap Pipeline infrastructure from a Kubernetes cluster

set -e

# Configuration
DEFAULT_NAMESPACE="reposentry-system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration variables
SYSTEM_NAMESPACE="${BOOTSTRAP_NAMESPACE:-$DEFAULT_NAMESPACE}"
DRY_RUN="${BOOTSTRAP_DRY_RUN:-false}"
FORCE="${BOOTSTRAP_FORCE:-false}"
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

Uninstall RepoSentry Bootstrap Pipeline infrastructure from Kubernetes cluster.

OPTIONS:
    -h, --help                    Show this help message
    -n, --namespace NAMESPACE     System namespace (default: $DEFAULT_NAMESPACE)
    -d, --dry-run                Perform dry run without actual deletion
    -f, --force                  Force deletion without confirmation
    -v, --verbose                Enable verbose output
    --keep-namespace             Keep the system namespace after cleanup

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE          Override system namespace
    BOOTSTRAP_DRY_RUN           Set to 'true' for dry run
    BOOTSTRAP_FORCE             Set to 'true' to skip confirmation
    BOOTSTRAP_VERBOSE           Set to 'true' for verbose output

EXAMPLES:
    # Basic uninstallation (with confirmation)
    $0

    # Uninstall custom namespace
    $0 --namespace my-reposentry-system

    # Dry run to see what would be deleted
    $0 --dry-run

    # Force uninstall without confirmation
    $0 --force

EOF
}

# Function to confirm deletion
confirm_deletion() {
    if [ "$FORCE" = "true" ]; then
        print_warning "Force mode enabled - skipping confirmation"
        return 0
    fi

    echo
    print_warning "This will DELETE the following Bootstrap Pipeline resources:"
    echo "  • Bootstrap Pipeline: reposentry-bootstrap-pipeline"
    echo "  • Bootstrap Tasks: 5 tasks"
    echo "  • RBAC: ServiceAccount, ClusterRole, ClusterRoleBinding"
    if [ "$KEEP_NAMESPACE" != "true" ]; then
        echo "  • System Namespace: $SYSTEM_NAMESPACE"
    fi
    echo
    print_warning "Any running PipelineRuns will be terminated!"
    echo

    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Uninstall cancelled by user"
        exit 0
    fi
}

# Function to delete RBAC resources
delete_rbac_resources() {
    print_status "Deleting RBAC resources..."

    if [ "$DRY_RUN" = "true" ]; then
        DRY_RUN_FLAG="--dry-run=client"
    else
        DRY_RUN_FLAG=""
    fi

    # Delete ClusterRoleBinding
    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        print_verbose "Deleting ClusterRoleBinding..."
        kubectl delete clusterrolebinding reposentry-bootstrap-binding $DRY_RUN_FLAG
        print_success "ClusterRoleBinding deleted"
    else
        print_verbose "ClusterRoleBinding not found (already deleted)"
    fi

    # Delete ClusterRole
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_verbose "Deleting ClusterRole..."
        kubectl delete clusterrole reposentry-bootstrap-role $DRY_RUN_FLAG
        print_success "ClusterRole deleted"
    else
        print_verbose "ClusterRole not found (already deleted)"
    fi

    # Delete ServiceAccount
    if kubectl get serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Deleting ServiceAccount..."
        kubectl delete serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" $DRY_RUN_FLAG
        print_success "ServiceAccount deleted"
    else
        print_verbose "ServiceAccount not found (already deleted)"
    fi
}

# Function to delete Bootstrap Pipeline and Tasks
delete_pipeline_resources() {
    print_status "Deleting Bootstrap Pipeline and Tasks..."

    if [ "$DRY_RUN" = "true" ]; then
        DRY_RUN_FLAG="--dry-run=client"
    else
        DRY_RUN_FLAG=""
    fi

    # Delete Bootstrap Pipeline
    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Deleting Bootstrap Pipeline..."
        kubectl delete pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" $DRY_RUN_FLAG
        print_success "Bootstrap Pipeline deleted"
    else
        print_verbose "Bootstrap Pipeline not found (already deleted)"
    fi

    # Delete Bootstrap Tasks
    local tasks=(
        "reposentry-bootstrap-clone"
        "reposentry-bootstrap-compute-namespace"
        "reposentry-bootstrap-validate"
        "reposentry-bootstrap-ensure-namespace"
        "reposentry-bootstrap-apply"
    )

    for task in "${tasks[@]}"; do
        if kubectl get task "$task" -n "$SYSTEM_NAMESPACE" &> /dev/null; then
            print_verbose "Deleting Task: $task"
            kubectl delete task "$task" -n "$SYSTEM_NAMESPACE" $DRY_RUN_FLAG
        else
            print_verbose "Task $task not found (already deleted)"
        fi
    done
    print_success "Bootstrap Tasks deleted"
}

# Function to delete PipelineRuns
delete_pipeline_runs() {
    print_status "Deleting Bootstrap PipelineRuns..."

    if [ "$DRY_RUN" = "true" ]; then
        DRY_RUN_FLAG="--dry-run=client"
    else
        DRY_RUN_FLAG=""
    fi

    # Check for existing PipelineRuns
    local pipelinerun_count
    pipelinerun_count=$(kubectl get pipelineruns -n "$SYSTEM_NAMESPACE" --no-headers 2>/dev/null | grep "reposentry-bootstrap" | wc -l)

    if [ "$pipelinerun_count" -eq 0 ]; then
        print_verbose "No Bootstrap PipelineRuns found"
    else
        print_warning "Found $pipelinerun_count Bootstrap PipelineRuns - deleting..."
        kubectl delete pipelineruns -n "$SYSTEM_NAMESPACE" -l tekton.dev/pipeline=reposentry-bootstrap-pipeline $DRY_RUN_FLAG
        print_success "Bootstrap PipelineRuns deleted"
    fi
}

# Function to delete namespace
delete_namespace() {
    if [ "$KEEP_NAMESPACE" = "true" ]; then
        print_status "Keeping system namespace (--keep-namespace specified)"
        return
    fi

    print_status "Deleting system namespace..."

    if [ "$DRY_RUN" = "true" ]; then
        DRY_RUN_FLAG="--dry-run=client"
    else
        DRY_RUN_FLAG=""
    fi

    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Deleting namespace: $SYSTEM_NAMESPACE"
        kubectl delete namespace "$SYSTEM_NAMESPACE" $DRY_RUN_FLAG
        print_success "System namespace deleted"
    else
        print_verbose "System namespace not found (already deleted)"
    fi
}

# Function to verify deletion
verify_deletion() {
    if [ "$DRY_RUN" = "true" ]; then
        print_status "Skipping deletion verification (dry run mode)"
        return
    fi

    print_status "Verifying deletion..."

    # Check if resources are gone
    local remaining_resources=0

    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_error "Bootstrap Pipeline still exists"
        remaining_resources=$((remaining_resources + 1))
    fi

    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_error "Bootstrap ClusterRole still exists"
        remaining_resources=$((remaining_resources + 1))
    fi

    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        print_error "Bootstrap ClusterRoleBinding still exists"
        remaining_resources=$((remaining_resources + 1))
    fi

    if [ "$KEEP_NAMESPACE" != "true" ] && kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_error "System namespace still exists"
        remaining_resources=$((remaining_resources + 1))
    fi

    if [ "$remaining_resources" -eq 0 ]; then
        print_success "All resources successfully deleted"
    else
        print_warning "$remaining_resources resources may still exist"
        print_warning "Manual cleanup may be required"
    fi
}

# Function to show post-uninstall information
show_post_uninstall_info() {
    cat << EOF

${GREEN}🗑️  Bootstrap Pipeline Uninstall Completed!${NC}

${BLUE}📋 What was removed:${NC}
  • Bootstrap Pipeline: reposentry-bootstrap-pipeline
  • Bootstrap Tasks: 5 tasks
  • RBAC: ServiceAccount, ClusterRole, ClusterRoleBinding
EOF

    if [ "$KEEP_NAMESPACE" = "true" ]; then
        echo -e "  • System Namespace: ${YELLOW}KEPT${NC} (${SYSTEM_NAMESPACE})"
    else
        echo -e "  • System Namespace: ${SYSTEM_NAMESPACE}"
    fi

    cat << EOF

${BLUE}🔍 Verification Commands:${NC}
  # Check if resources are gone
  kubectl get pipeline,task -n ${SYSTEM_NAMESPACE}
  kubectl get clusterrole,clusterrolebinding | grep reposentry-bootstrap

${BLUE}📚 Next Steps:${NC}
  • To reinstall: Run ./scripts/install-bootstrap-pipeline.sh
  • Clean up any user namespaces if needed: kubectl get namespaces -l reposentry.io/managed=true

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
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -f|--force)
            FORCE="true"
            shift
            ;;
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        --keep-namespace)
            KEEP_NAMESPACE="true"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main uninstall flow
main() {
    echo -e "${RED}🗑️  Uninstalling RepoSentry Bootstrap Pipeline...${NC}"
    echo

    confirm_deletion
    delete_pipeline_runs
    delete_pipeline_resources
    delete_rbac_resources
    delete_namespace
    verify_deletion
    show_post_uninstall_info
}

# Run main function
main "$@"

