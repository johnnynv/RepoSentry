#!/bin/bash

# RepoSentry Bootstrap Pipeline Uninstallation Script
# This script removes the Bootstrap Pipeline infrastructure from Kubernetes cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
YAML_DIR="$SCRIPT_DIR/../"
DEFAULT_NAMESPACE="reposentry-system"
SYSTEM_NAMESPACE="${BOOTSTRAP_NAMESPACE:-$DEFAULT_NAMESPACE}"
DRY_RUN="${BOOTSTRAP_DRY_RUN:-false}"
VERBOSE="${BOOTSTRAP_VERBOSE:-false}"
FORCE="${BOOTSTRAP_FORCE:-false}"

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
This script removes all Bootstrap Pipeline resources including Pipeline, Tasks, and RBAC.

OPTIONS:
    -h, --help                    Show this help message
    -n, --namespace NAMESPACE     System namespace (default: $DEFAULT_NAMESPACE)
    -d, --dry-run                Perform dry run without actual deletion
    -v, --verbose                Enable verbose output
    --force                      Force deletion without confirmation prompts

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE          Override system namespace
    BOOTSTRAP_DRY_RUN           Set to 'true' for dry run
    BOOTSTRAP_VERBOSE           Set to 'true' for verbose output
    BOOTSTRAP_FORCE             Set to 'true' to skip confirmation

EXAMPLES:
    # Basic uninstallation (with confirmation prompt)
    $0

    # Uninstall from custom namespace
    $0 --namespace my-reposentry-system

    # Dry run to see what would be deleted
    $0 --dry-run

    # Force uninstall without confirmation
    $0 --force

    # Verbose uninstallation
    $0 --verbose

CAUTION:
    This will permanently delete all Bootstrap Pipeline resources!
    Make sure no PipelineRuns are currently executing.

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

    print_success "Prerequisites check completed"
}

# Function to check if resources exist
check_resources_exist() {
    print_status "Checking if Bootstrap Pipeline resources exist..."

    local resources_found=false

    # Check namespace
    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Found namespace: $SYSTEM_NAMESPACE"
        resources_found=true
    fi

    # Check pipeline
    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Found Bootstrap Pipeline"
        resources_found=true
    fi

    # Check tasks
    local task_count=$(kubectl get tasks -n "$SYSTEM_NAMESPACE" -l reposentry.io/component=bootstrap --no-headers 2>/dev/null | wc -l)
    if [ "$task_count" -gt 0 ]; then
        print_verbose "Found $task_count Bootstrap Tasks"
        resources_found=true
    fi

    # Check RBAC
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_verbose "Found Bootstrap ClusterRole"
        resources_found=true
    fi

    # Check Tekton Triggers components
    if kubectl get eventlistener reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Found Bootstrap EventListener"
        resources_found=true
    fi

    if [ "$resources_found" = "false" ]; then
        print_warning "No Bootstrap Pipeline resources found"
        print_warning "Nothing to uninstall"
        exit 0
    fi

    print_success "Bootstrap Pipeline resources found"
}

# Function to show what will be deleted
show_deletion_plan() {
    print_status "The following resources will be deleted:"
    echo
    
    # Check and list resources
    echo "📋 Resources to be removed:"
    
    # Namespace resources
    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • Namespace: $SYSTEM_NAMESPACE"
    fi
    
    # Pipeline and Tasks
    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • Pipeline: reposentry-bootstrap-pipeline"
    fi
    
    local tasks=$(kubectl get tasks -n "$SYSTEM_NAMESPACE" -l reposentry.io/component=bootstrap --no-headers 2>/dev/null | awk '{print $1}' || echo "")
    if [ -n "$tasks" ]; then
        echo "   • Tasks:"
        for task in $tasks; do
            echo "     - $task"
        done
    fi
    
    # RBAC
    if kubectl get serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • ServiceAccount: reposentry-bootstrap-sa"
    fi
    
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        echo "   • ClusterRole: reposentry-bootstrap-role"
    fi
    
    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        echo "   • ClusterRoleBinding: reposentry-bootstrap-binding"
    fi
    
    # Tekton Triggers components
    if kubectl get triggerbinding reposentry-bootstrap-binding -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • TriggerBinding: reposentry-bootstrap-binding"
    fi
    
    if kubectl get triggertemplate reposentry-bootstrap-template -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • TriggerTemplate: reposentry-bootstrap-template"
    fi
    
    if kubectl get eventlistener reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • EventListener: reposentry-standard-eventlistener"
    fi
    
    if kubectl get service el-reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • EventListener Service: el-reposentry-standard-eventlistener"
    fi
    
    if kubectl get ingress reposentry-eventlistener-ingress -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        echo "   • Ingress: reposentry-eventlistener-ingress"
    fi
    
    echo
}

# Function to confirm deletion
confirm_deletion() {
    if [ "$FORCE" = "true" ]; then
        print_warning "Force mode enabled - skipping confirmation"
        return 0
    fi

    echo -e "${YELLOW}⚠️  This will permanently delete all Bootstrap Pipeline resources!${NC}"
    echo -e "${YELLOW}⚠️  Make sure no PipelineRuns are currently executing.${NC}"
    echo
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Uninstallation cancelled by user"
        exit 0
    fi
}

# Function to check for running PipelineRuns
check_running_pipelineruns() {
    print_status "Checking for running PipelineRuns..."
    
    local running_runs=$(kubectl get pipelineruns -n "$SYSTEM_NAMESPACE" --field-selector=status.conditions[-1].reason!=Succeeded,status.conditions[-1].reason!=Failed,status.conditions[-1].reason!=Cancelled --no-headers 2>/dev/null | wc -l || echo "0")
    
    if [ "$running_runs" -gt 0 ]; then
        print_warning "Found $running_runs running PipelineRuns in namespace $SYSTEM_NAMESPACE"
        
        if [ "$FORCE" = "false" ]; then
            echo
            read -p "Do you want to wait for them to complete? (y/N): " -n 1 -r
            echo
            
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                print_status "Waiting for PipelineRuns to complete..."
                kubectl wait --for=condition=Succeeded pipelineruns --all -n "$SYSTEM_NAMESPACE" --timeout=300s || true
            else
                print_warning "Proceeding with uninstall despite running PipelineRuns"
            fi
        else
            print_warning "Force mode - proceeding despite running PipelineRuns"
        fi
    else
        print_success "No running PipelineRuns found"
    fi
}

# Function to uninstall Bootstrap Pipeline
uninstall_bootstrap_pipeline() {
    print_status "Uninstalling Bootstrap Pipeline infrastructure..."

    if [ "$DRY_RUN" = "true" ]; then
        print_warning "DRY RUN MODE - No actual changes will be made"
        DRY_RUN_FLAG="--dry-run=server"
    else
        DRY_RUN_FLAG=""
    fi

    # Delete resources in reverse order (opposite of installation)
    local files=(
        "11-ingress.yaml"
        "10-service.yaml"
        "09-eventlistener.yaml"
        "08-triggertemplate.yaml"
        "07-triggerbinding.yaml"
        "06-pipeline.yaml"
        "05-tasks.yaml"
        "04-clusterrolebinding.yaml"
        "03-clusterrole.yaml"
        "02-serviceaccount.yaml"
        "01-namespace.yaml"
    )

    for file in "${files[@]}"; do
        if [ -f "$YAML_DIR/$file" ]; then
            print_status "Deleting resources from $file..."
            kubectl delete -f "$YAML_DIR/$file" $DRY_RUN_FLAG --ignore-not-found=true
            if [ $? -eq 0 ]; then
                print_verbose "Successfully deleted resources from $file"
            else
                print_warning "Some resources from $file may not exist (this is normal)"
            fi
        else
            print_warning "File $file not found, skipping"
        fi
    done

    if [ "$DRY_RUN" = "true" ]; then
        print_success "DRY RUN completed - Bootstrap Pipeline would be uninstalled"
    else
        print_success "Bootstrap Pipeline uninstalled successfully"
    fi
}

# Function to verify uninstallation
verify_uninstallation() {
    if [ "$DRY_RUN" = "true" ]; then
        print_status "Skipping verification (dry run mode)"
        return
    fi

    print_status "Verifying uninstallation..."

    local resources_remaining=false

    # Check if namespace still exists
    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_warning "Namespace '$SYSTEM_NAMESPACE' still exists"
        resources_remaining=true
    else
        print_verbose "Namespace '$SYSTEM_NAMESPACE' successfully removed"
    fi

    # Check RBAC
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_warning "ClusterRole 'reposentry-bootstrap-role' still exists"
        resources_remaining=true
    else
        print_verbose "ClusterRole successfully removed"
    fi

    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        print_warning "ClusterRoleBinding 'reposentry-bootstrap-binding' still exists"
        resources_remaining=true
    else
        print_verbose "ClusterRoleBinding successfully removed"
    fi

    if [ "$resources_remaining" = "true" ]; then
        print_warning "Some resources may still exist - this could be normal during cleanup"
        print_warning "You may need to manually clean up remaining resources"
    else
        print_success "All Bootstrap Pipeline resources successfully removed"
    fi
}

# Function to show post-uninstall information
show_post_uninstall_info() {
    echo
    echo -e "${GREEN}🗑️  Bootstrap Pipeline Uninstallation Completed!${NC}"
    echo
    echo -e "${BLUE}📋 Summary:${NC}"
    echo "  • System Namespace: ${SYSTEM_NAMESPACE} (removed)"
    echo "  • Bootstrap Pipeline: reposentry-bootstrap-pipeline (removed)"
    echo "  • Bootstrap Tasks: All tasks removed"
    echo "  • RBAC: ClusterRole and ClusterRoleBinding removed"
    echo
    echo -e "${BLUE}🔍 Verification Commands:${NC}"
    echo "  # Check if namespace was removed"
    echo "  kubectl get namespace ${SYSTEM_NAMESPACE}"
    echo
    echo "  # Check if RBAC was removed"
    echo "  kubectl get clusterrole,clusterrolebinding | grep reposentry-bootstrap"
    echo
    echo -e "${BLUE}📚 Next Steps:${NC}"
    echo "  • RepoSentry will no longer be able to process Tekton resources"
    echo "  • To reinstall: Run './install.sh' in this directory"
    echo "  • To completely remove RepoSentry: Also remove application configuration"
    echo
    echo -e "${BLUE}💡 Note:${NC}"
    echo "  User namespaces created by the Bootstrap Pipeline are NOT automatically removed."
    echo "  If you want to clean up user namespaces, list them with:"
    echo "  kubectl get namespaces -l reposentry.io/managed=true"
    echo
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

# Main uninstallation flow
main() {
    echo -e "${RED}🗑️  Uninstalling RepoSentry Bootstrap Pipeline...${NC}"
    echo -e "${BLUE}📁 Working directory: $SCRIPT_DIR${NC}"
    echo

    check_prerequisites
    check_resources_exist
    show_deletion_plan
    confirm_deletion
    check_running_pipelineruns
    uninstall_bootstrap_pipeline
    verify_uninstallation
    show_post_uninstall_info
}

# Run main function
main "$@"
