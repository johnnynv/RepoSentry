#!/bin/bash

# RepoSentry Bootstrap Pipeline Validation Script
# This script validates that the Bootstrap Pipeline infrastructure is properly deployed

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

Validate RepoSentry Bootstrap Pipeline deployment in Kubernetes cluster.

OPTIONS:
    -h, --help                    Show this help message
    -n, --namespace NAMESPACE     System namespace (default: $DEFAULT_NAMESPACE)
    -v, --verbose                Enable verbose output
    --detailed                   Show detailed resource information

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE          Override system namespace
    BOOTSTRAP_VERBOSE           Set to 'true' for verbose output

EXAMPLES:
    # Basic validation
    $0

    # Validate custom namespace
    $0 --namespace my-reposentry-system

    # Detailed validation with verbose output
    $0 --verbose --detailed

EOF
}

# Validation counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# Function to run a validation check
run_check() {
    local check_name="$1"
    local check_command="$2"
    local success_message="$3"
    local error_message="$4"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    print_verbose "Running check: $check_name"

    if eval "$check_command" &> /dev/null; then
        print_success "$success_message"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        print_error "$error_message"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# Function to validate cluster connectivity
validate_cluster_connectivity() {
    print_status "Validating cluster connectivity..."

    run_check "kubectl_available" \
        "command -v kubectl" \
        "kubectl is available" \
        "kubectl is not installed or not in PATH"

    run_check "cluster_reachable" \
        "kubectl cluster-info" \
        "Kubernetes cluster is reachable" \
        "Cannot connect to Kubernetes cluster"

    run_check "tekton_api" \
        "kubectl api-resources | grep -q 'tekton.dev'" \
        "Tekton APIs are available" \
        "Tekton APIs not found - ensure Tekton Pipelines is installed"
}

# Function to validate namespace
validate_namespace() {
    print_status "Validating system namespace..."

    run_check "namespace_exists" \
        "kubectl get namespace '$SYSTEM_NAMESPACE'" \
        "System namespace '$SYSTEM_NAMESPACE' exists" \
        "System namespace '$SYSTEM_NAMESPACE' not found"

    if [ "$VERBOSE" = "true" ] && kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Namespace details:"
        kubectl get namespace "$SYSTEM_NAMESPACE" -o yaml | grep -E "(name:|labels:|annotations:)" | sed 's/^/  /'
    fi
}

# Function to validate Bootstrap Pipeline
validate_bootstrap_pipeline() {
    print_status "Validating Bootstrap Pipeline..."

    run_check "pipeline_exists" \
        "kubectl get pipeline reposentry-bootstrap-pipeline -n '$SYSTEM_NAMESPACE'" \
        "Bootstrap Pipeline exists" \
        "Bootstrap Pipeline not found"

    if [ "$VERBOSE" = "true" ] && kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_verbose "Pipeline tasks:"
        kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" -o jsonpath='{.spec.tasks[*].name}' | tr ' ' '\n' | sed 's/^/  - /'
    fi
}

# Function to validate Bootstrap Tasks
validate_bootstrap_tasks() {
    print_status "Validating Bootstrap Tasks..."

    local expected_tasks=(
        "reposentry-bootstrap-clone"
        "reposentry-bootstrap-compute-namespace"
        "reposentry-bootstrap-validate"
        "reposentry-bootstrap-ensure-namespace"
        "reposentry-bootstrap-apply"
    )

    for task in "${expected_tasks[@]}"; do
        run_check "task_${task}" \
            "kubectl get task '$task' -n '$SYSTEM_NAMESPACE'" \
            "Task '$task' exists" \
            "Task '$task' not found"
    done

    # Count total tasks
    local task_count
    task_count=$(kubectl get tasks -n "$SYSTEM_NAMESPACE" --no-headers 2>/dev/null | grep "reposentry-bootstrap" | wc -l)
    print_verbose "Total Bootstrap Tasks found: $task_count"
}

# Function to validate RBAC resources
validate_rbac() {
    print_status "Validating RBAC resources..."

    run_check "serviceaccount_exists" \
        "kubectl get serviceaccount reposentry-bootstrap-sa -n '$SYSTEM_NAMESPACE'" \
        "Bootstrap ServiceAccount exists" \
        "Bootstrap ServiceAccount not found"

    run_check "clusterrole_exists" \
        "kubectl get clusterrole reposentry-bootstrap-role" \
        "Bootstrap ClusterRole exists" \
        "Bootstrap ClusterRole not found"

    run_check "clusterrolebinding_exists" \
        "kubectl get clusterrolebinding reposentry-bootstrap-binding" \
        "Bootstrap ClusterRoleBinding exists" \
        "Bootstrap ClusterRoleBinding not found"

    # Validate role permissions
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        run_check "role_has_namespace_permissions" \
            "kubectl get clusterrole reposentry-bootstrap-role -o jsonpath='{.rules[*].resources}' | grep -q namespaces" \
            "ClusterRole has namespace permissions" \
            "ClusterRole missing namespace permissions"

        run_check "role_has_tekton_permissions" \
            "kubectl get clusterrole reposentry-bootstrap-role -o jsonpath='{.rules[*].resources}' | grep -q pipelines" \
            "ClusterRole has Tekton permissions" \
            "ClusterRole missing Tekton permissions"
    fi
}

# Function to validate recent PipelineRuns (if any)
validate_pipeline_runs() {
    print_status "Checking for recent PipelineRuns..."

    local pipelinerun_count
    pipelinerun_count=$(kubectl get pipelineruns -n "$SYSTEM_NAMESPACE" --no-headers 2>/dev/null | grep "reposentry-bootstrap" | wc -l)

    if [ "$pipelinerun_count" -eq 0 ]; then
        print_warning "No Bootstrap PipelineRuns found (this is normal for a fresh installation)"
    else
        print_success "Found $pipelinerun_count Bootstrap PipelineRuns"
        
        if [ "$VERBOSE" = "true" ]; then
            print_verbose "Recent PipelineRuns:"
            kubectl get pipelineruns -n "$SYSTEM_NAMESPACE" --no-headers | grep "reposentry-bootstrap" | head -5 | sed 's/^/  /'
        fi
    fi
}

# Function to check resource health
validate_resource_health() {
    print_status "Validating resource health..."

    # Check if pipeline is valid (no syntax errors)
    run_check "pipeline_valid" \
        "kubectl get pipeline reposentry-bootstrap-pipeline -n '$SYSTEM_NAMESPACE' -o jsonpath='{.status}' | grep -v 'conditions.*False'" \
        "Bootstrap Pipeline is healthy" \
        "Bootstrap Pipeline may have validation errors"

    # Check if all tasks are valid
    local invalid_tasks
    invalid_tasks=$(kubectl get tasks -n "$SYSTEM_NAMESPACE" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.conditions[?(@.type=="Succeeded")].status}{"\n"}{end}' 2>/dev/null | grep "False" | wc -l)
    
    if [ "$invalid_tasks" -eq 0 ]; then
        print_success "All Bootstrap Tasks are healthy"
    else
        print_warning "$invalid_tasks Bootstrap Tasks may have validation errors"
    fi
}

# Function to show detailed resource information
show_detailed_info() {
    if [ "$DETAILED" != "true" ]; then
        return
    fi

    print_status "Detailed resource information:"

    echo
    echo -e "${BLUE}📋 Namespace Information:${NC}"
    kubectl describe namespace "$SYSTEM_NAMESPACE" 2>/dev/null | head -20

    echo
    echo -e "${BLUE}📋 Pipeline Information:${NC}"
    kubectl describe pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" 2>/dev/null | head -30

    echo
    echo -e "${BLUE}📋 Tasks Summary:${NC}"
    kubectl get tasks -n "$SYSTEM_NAMESPACE" -o wide 2>/dev/null | grep "reposentry-bootstrap"

    echo
    echo -e "${BLUE}📋 RBAC Summary:${NC}"
    echo "ServiceAccount:"
    kubectl get serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" 2>/dev/null
    echo "ClusterRole:"
    kubectl get clusterrole reposentry-bootstrap-role 2>/dev/null
    echo "ClusterRoleBinding:"
    kubectl get clusterrolebinding reposentry-bootstrap-binding 2>/dev/null
}

# Function to show validation summary
show_validation_summary() {
    echo
    echo -e "${BLUE}📊 Validation Summary:${NC}"
    echo "  Total Checks: $TOTAL_CHECKS"
    echo -e "  Passed: ${GREEN}$PASSED_CHECKS${NC}"
    
    if [ "$FAILED_CHECKS" -gt 0 ]; then
        echo -e "  Failed: ${RED}$FAILED_CHECKS${NC}"
        echo
        print_error "Bootstrap Pipeline validation FAILED"
        print_error "Please review the errors above and ensure proper installation"
        return 1
    else
        echo -e "  Failed: ${GREEN}0${NC}"
        echo
        print_success "Bootstrap Pipeline validation PASSED"
        print_success "All components are properly deployed and healthy"
        return 0
    fi
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
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        --detailed)
            DETAILED="true"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main validation flow
main() {
    echo -e "${GREEN}🔍 Validating RepoSentry Bootstrap Pipeline...${NC}"
    echo

    validate_cluster_connectivity
    validate_namespace
    validate_bootstrap_pipeline
    validate_bootstrap_tasks
    validate_rbac
    validate_pipeline_runs
    validate_resource_health
    show_detailed_info
    show_validation_summary
}

# Run main function
main "$@"

