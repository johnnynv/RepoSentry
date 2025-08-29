#!/bin/bash

# RepoSentry Bootstrap Pipeline Validation Script
# This script validates the Bootstrap Pipeline installation and checks health status

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
VERBOSE="${BOOTSTRAP_VERBOSE:-false}"
CHECK_CONNECTIVITY="${BOOTSTRAP_CHECK_CONNECTIVITY:-true}"

# Validation results
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

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

print_check_result() {
    local status="$1"
    local message="$2"
    local details="$3"
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    case "$status" in
        "PASS")
            echo -e "  ✅ $message"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            ;;
        "FAIL")
            echo -e "  ❌ $message"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            if [ -n "$details" ]; then
                echo -e "     ${RED}$details${NC}"
            fi
            ;;
        "WARN")
            echo -e "  ⚠️  $message"
            WARNING_CHECKS=$((WARNING_CHECKS + 1))
            if [ -n "$details" ]; then
                echo -e "     ${YELLOW}$details${NC}"
            fi
            ;;
    esac
    
    if [ "$VERBOSE" = "true" ] && [ -n "$details" ]; then
        echo -e "     ${BLUE}Details: $details${NC}"
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Validate RepoSentry Bootstrap Pipeline installation and check health status.
This script performs comprehensive validation of all Bootstrap Pipeline components.

OPTIONS:
    -h, --help                    Show this help message
    -n, --namespace NAMESPACE     System namespace (default: $DEFAULT_NAMESPACE)
    -v, --verbose                Enable verbose output
    --no-connectivity            Skip connectivity tests
    --quick                      Run only basic validation checks

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE           Override system namespace
    BOOTSTRAP_VERBOSE            Set to 'true' for verbose output
    BOOTSTRAP_CHECK_CONNECTIVITY Set to 'false' to skip connectivity tests

EXAMPLES:
    # Basic validation
    $0

    # Validate custom namespace
    $0 --namespace my-reposentry-system

    # Verbose validation with detailed output
    $0 --verbose

    # Quick validation (skip connectivity tests)
    $0 --quick

    # Check specific namespace without connectivity tests
    $0 --namespace production-reposentry --no-connectivity

EOF
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_check_result "FAIL" "kubectl command" "kubectl is not installed or not in PATH"
        exit 1
    else
        print_check_result "PASS" "kubectl command available"
    fi

    # Check if we can connect to Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_check_result "FAIL" "Kubernetes cluster connectivity" "Cannot connect to cluster"
        exit 1
    else
        print_check_result "PASS" "Kubernetes cluster connectivity"
    fi

    # Check if Tekton is installed
    if ! kubectl api-resources | grep -q "tekton.dev"; then
        print_check_result "FAIL" "Tekton Pipelines installation" "Tekton API resources not found"
        exit 1
    else
        print_check_result "PASS" "Tekton Pipelines installation"
    fi
}

# Function to validate namespace
validate_namespace() {
    print_status "Validating namespace..."

    # Check if namespace exists
    if kubectl get namespace "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "Namespace '$SYSTEM_NAMESPACE' exists"
        
        # Check namespace labels
        local labels=$(kubectl get namespace "$SYSTEM_NAMESPACE" -o jsonpath='{.metadata.labels}' 2>/dev/null || echo "{}")
        if echo "$labels" | grep -q "reposentry.io/component"; then
            print_check_result "PASS" "Namespace has correct labels"
        else
            print_check_result "WARN" "Namespace missing RepoSentry labels" "Consider adding reposentry.io/component label"
        fi
    else
        print_check_result "FAIL" "Namespace '$SYSTEM_NAMESPACE' exists" "Namespace not found"
        return 1
    fi
}

# Function to validate pipeline
validate_pipeline() {
    print_status "Validating Bootstrap Pipeline..."

    # Check if pipeline exists
    if kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "Bootstrap Pipeline exists"
        
        # Check pipeline status and configuration
        local pipeline_spec=$(kubectl get pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" -o yaml 2>/dev/null)
        
        # Check if pipeline has required tasks
        local task_count=$(echo "$pipeline_spec" | grep -c "taskRef:" || echo "0")
        if [ "$task_count" -ge 5 ]; then
            print_check_result "PASS" "Pipeline has required tasks ($task_count tasks)"
        else
            print_check_result "WARN" "Pipeline task count" "Expected 5+ tasks, found $task_count"
        fi
        
        # Check required parameters
        if echo "$pipeline_spec" | grep -q "repo-url"; then
            print_check_result "PASS" "Pipeline has required parameters"
        else
            print_check_result "FAIL" "Pipeline parameters" "Missing required repo-url parameter"
        fi
    else
        print_check_result "FAIL" "Bootstrap Pipeline exists" "Pipeline 'reposentry-bootstrap-pipeline' not found"
    fi
}

# Function to validate tasks
validate_tasks() {
    print_status "Validating Bootstrap Tasks..."

    local expected_tasks=(
        "reposentry-bootstrap-clone"
        "reposentry-bootstrap-compute-namespace"
        "reposentry-bootstrap-validate"
        "reposentry-bootstrap-ensure-namespace"
        "reposentry-bootstrap-apply"
        "reposentry-bootstrap-run"
    )

    local found_tasks=0

    for task in "${expected_tasks[@]}"; do
        if kubectl get task "$task" -n "$SYSTEM_NAMESPACE" &> /dev/null; then
            print_check_result "PASS" "Task '$task' exists"
            found_tasks=$((found_tasks + 1))
        else
            print_check_result "FAIL" "Task '$task' exists" "Task not found"
        fi
    done

    if [ "$found_tasks" -eq "${#expected_tasks[@]}" ]; then
        print_check_result "PASS" "All Bootstrap Tasks are available ($found_tasks/${#expected_tasks[@]})"
    else
        print_check_result "FAIL" "Bootstrap Tasks completeness" "Found $found_tasks/${#expected_tasks[@]} tasks"
    fi
}

# Function to validate Tekton Triggers components
validate_tekton_triggers() {
    print_status "Validating Tekton Triggers components..."

    # Check TriggerBinding
    if kubectl get triggerbinding reposentry-bootstrap-binding -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "TriggerBinding 'reposentry-bootstrap-binding' exists"
        
        # Check if TriggerBinding has required parameters
        local binding_spec=$(kubectl get triggerbinding reposentry-bootstrap-binding -n "$SYSTEM_NAMESPACE" -o yaml 2>/dev/null)
        if echo "$binding_spec" | grep -q "repo-full-name"; then
            print_check_result "PASS" "TriggerBinding has required parameters"
        else
            print_check_result "WARN" "TriggerBinding parameters" "May be missing repo-full-name parameter"
        fi
    else
        print_check_result "FAIL" "TriggerBinding exists" "reposentry-bootstrap-binding not found"
    fi

    # Check TriggerTemplate
    if kubectl get triggertemplate reposentry-bootstrap-template -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "TriggerTemplate 'reposentry-bootstrap-template' exists"
        
        # Check if TriggerTemplate references the correct pipeline
        local template_spec=$(kubectl get triggertemplate reposentry-bootstrap-template -n "$SYSTEM_NAMESPACE" -o yaml 2>/dev/null)
        if echo "$template_spec" | grep -q "reposentry-bootstrap-pipeline"; then
            print_check_result "PASS" "TriggerTemplate references correct pipeline"
        else
            print_check_result "FAIL" "TriggerTemplate configuration" "Does not reference reposentry-bootstrap-pipeline"
        fi
    else
        print_check_result "FAIL" "TriggerTemplate exists" "reposentry-bootstrap-template not found"
    fi

    # Check EventListener
    if kubectl get eventlistener reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "EventListener 'reposentry-standard-eventlistener' exists"
        
        # Check EventListener status
        local el_status=$(kubectl get eventlistener reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" -o jsonpath='{.status.conditions[0].status}' 2>/dev/null || echo "Unknown")
        if [ "$el_status" = "True" ]; then
            print_check_result "PASS" "EventListener is ready"
        else
            print_check_result "WARN" "EventListener status" "Status: $el_status"
        fi
        
        # Check if EventListener service was created
        if kubectl get service el-reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
            print_check_result "PASS" "EventListener service exists"
        else
            print_check_result "FAIL" "EventListener service" "Service el-reposentry-standard-eventlistener not found"
        fi
    else
        print_check_result "FAIL" "EventListener exists" "reposentry-standard-eventlistener not found"
    fi

    # Check Ingress
    if kubectl get ingress reposentry-eventlistener-ingress -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "EventListener Ingress exists"
        
        # Check Ingress configuration
        local ingress_class=$(kubectl get ingress reposentry-eventlistener-ingress -n "$SYSTEM_NAMESPACE" -o jsonpath='{.spec.ingressClassName}' 2>/dev/null || echo "none")
        if [ "$ingress_class" != "none" ]; then
            print_check_result "PASS" "Ingress has ingressClassName: $ingress_class"
        else
            print_check_result "WARN" "Ingress configuration" "No ingressClassName specified"
        fi
        
        # Get webhook URL
        local webhook_host=$(kubectl get ingress reposentry-eventlistener-ingress -n "$SYSTEM_NAMESPACE" -o jsonpath='{.spec.rules[0].host}' 2>/dev/null || echo "unknown")
        if [ "$webhook_host" != "unknown" ]; then
            print_check_result "PASS" "Webhook URL available: http://$webhook_host/"
        else
            print_check_result "WARN" "Webhook URL" "Could not determine webhook URL"
        fi
    else
        print_check_result "FAIL" "EventListener Ingress exists" "reposentry-eventlistener-ingress not found"
    fi
}

# Function to validate RBAC
validate_rbac() {
    print_status "Validating RBAC configuration..."

    # Check ServiceAccount
    if kubectl get serviceaccount reposentry-bootstrap-sa -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "ServiceAccount 'reposentry-bootstrap-sa' exists"
    else
        print_check_result "FAIL" "ServiceAccount exists" "reposentry-bootstrap-sa not found"
    fi

    # Check ClusterRole
    if kubectl get clusterrole reposentry-bootstrap-role &> /dev/null; then
        print_check_result "PASS" "ClusterRole 'reposentry-bootstrap-role' exists"
        
        # Check role permissions
        local rules=$(kubectl get clusterrole reposentry-bootstrap-role -o jsonpath='{.rules}' 2>/dev/null)
        if echo "$rules" | grep -q "namespaces"; then
            print_check_result "PASS" "ClusterRole has namespace permissions"
        else
            print_check_result "WARN" "ClusterRole permissions" "May be missing namespace management permissions"
        fi
    else
        print_check_result "FAIL" "ClusterRole exists" "reposentry-bootstrap-role not found"
    fi

    # Check ClusterRoleBinding
    if kubectl get clusterrolebinding reposentry-bootstrap-binding &> /dev/null; then
        print_check_result "PASS" "ClusterRoleBinding 'reposentry-bootstrap-binding' exists"
        
        # Verify binding links SA to Role
        local subject=$(kubectl get clusterrolebinding reposentry-bootstrap-binding -o jsonpath='{.subjects[0].name}' 2>/dev/null)
        if [ "$subject" = "reposentry-bootstrap-sa" ]; then
            print_check_result "PASS" "ClusterRoleBinding correctly links ServiceAccount"
        else
            print_check_result "FAIL" "ClusterRoleBinding configuration" "ServiceAccount link incorrect"
        fi
    else
        print_check_result "FAIL" "ClusterRoleBinding exists" "reposentry-bootstrap-binding not found"
    fi
}

# Function to validate resource quotas
validate_resource_limits() {
    print_status "Validating resource configuration..."

    # Check if namespace has resource quotas (optional)
    local quota_count=$(kubectl get resourcequotas -n "$SYSTEM_NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$quota_count" -gt 0 ]; then
        print_check_result "PASS" "Resource quotas configured ($quota_count quotas)"
    else
        print_check_result "WARN" "Resource quotas" "No resource quotas found (optional but recommended)"
    fi

    # Check if there are any limit ranges
    local limits_count=$(kubectl get limitranges -n "$SYSTEM_NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$limits_count" -gt 0 ]; then
        print_check_result "PASS" "Limit ranges configured ($limits_count ranges)"
    else
        print_check_result "WARN" "Limit ranges" "No limit ranges found (optional but recommended)"
    fi
}

# Function to test pipeline connectivity
test_pipeline_connectivity() {
    if [ "$CHECK_CONNECTIVITY" = "false" ]; then
        print_status "Skipping connectivity tests (disabled)"
        return
    fi

    print_status "Testing pipeline connectivity..."

    # Check if we can describe the pipeline (permissions test)
    if kubectl describe pipeline reposentry-bootstrap-pipeline -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_check_result "PASS" "Pipeline description access"
    else
        print_check_result "FAIL" "Pipeline access permissions" "Cannot describe pipeline"
    fi

    # Check if ServiceAccount can create PipelineRuns (dry-run test)
    local test_pipelinerun=$(cat <<EOF
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: test-validation-run
  namespace: $SYSTEM_NAMESPACE
spec:
  serviceAccountName: reposentry-bootstrap-sa
  pipelineRef:
    name: reposentry-bootstrap-pipeline
  params:
  - name: repo-url
    value: "https://github.com/test/test.git"
  - name: commit-sha
    value: "test-sha"
  - name: target-namespace
    value: "test-namespace"
  workspaces:
  - name: source-workspace
    emptyDir: {}
  - name: tekton-workspace
    emptyDir: {}
EOF
)

    if echo "$test_pipelinerun" | kubectl apply --dry-run=client -f - &> /dev/null; then
        print_check_result "PASS" "PipelineRun creation permissions"
    else
        print_check_result "WARN" "PipelineRun creation test" "Dry-run validation failed (may indicate RBAC issues)"
    fi
}

# Function to check recent PipelineRuns
check_recent_activity() {
    print_status "Checking recent activity..."

    # Check for recent PipelineRuns
    local recent_runs=$(kubectl get pipelineruns -n "$SYSTEM_NAMESPACE" --sort-by=.metadata.creationTimestamp --no-headers 2>/dev/null | tail -5 || echo "")
    
    if [ -n "$recent_runs" ]; then
        local run_count=$(echo "$recent_runs" | wc -l)
        print_check_result "PASS" "Recent PipelineRun activity found ($run_count recent runs)"
        
        if [ "$VERBOSE" = "true" ]; then
            echo "     Recent PipelineRuns:"
            echo "$recent_runs" | while read line; do
                echo "       $line"
            done
        fi
    else
        print_check_result "WARN" "PipelineRun activity" "No recent PipelineRuns found (normal for new installation)"
    fi
}

# Function to validate against static files
validate_against_static_files() {
    print_status "Validating against static YAML files..."

    local yaml_files=(
        "01-namespace.yaml"
        "02-serviceaccount.yaml"
        "03-clusterrole.yaml"
        "04-clusterrolebinding.yaml"
        "05-tasks.yaml"
        "06-pipeline.yaml"
        "07-triggerbinding.yaml"
        "08-triggertemplate.yaml"
        "09-eventlistener.yaml"
        "10-service.yaml"
        "11-ingress.yaml"
    )

    local files_found=0

    for file in "${yaml_files[@]}"; do
        if [ -f "$YAML_DIR/$file" ]; then
            print_check_result "PASS" "Static file '$file' available"
            files_found=$((files_found + 1))
        else
            print_check_result "WARN" "Static file '$file' missing" "File not found in deployment directory"
        fi
    done

    if [ "$files_found" -eq "${#yaml_files[@]}" ]; then
        print_check_result "PASS" "All static YAML files are available"
    else
        print_check_result "WARN" "Static file completeness" "Found $files_found/${#yaml_files[@]} files"
    fi
}

# Function to show validation summary
show_validation_summary() {
    echo
    echo "=================================================================="
    echo -e "${BLUE}🔍 Bootstrap Pipeline Validation Summary${NC}"
    echo "=================================================================="
    echo
    echo -e "📊 ${BLUE}Validation Results:${NC}"
    echo -e "   • Total Checks: $TOTAL_CHECKS"
    echo -e "   • ✅ Passed: ${GREEN}$PASSED_CHECKS${NC}"
    echo -e "   • ❌ Failed: ${RED}$FAILED_CHECKS${NC}"
    echo -e "   • ⚠️  Warnings: ${YELLOW}$WARNING_CHECKS${NC}"
    echo

    # Overall status
    if [ "$FAILED_CHECKS" -eq 0 ]; then
        if [ "$WARNING_CHECKS" -eq 0 ]; then
            echo -e "🎉 ${GREEN}Overall Status: EXCELLENT${NC}"
            echo -e "   Bootstrap Pipeline is fully operational and ready to use!"
        else
            echo -e "✅ ${GREEN}Overall Status: GOOD${NC}"
            echo -e "   Bootstrap Pipeline is operational with minor recommendations."
        fi
    else
        echo -e "💥 ${RED}Overall Status: ISSUES DETECTED${NC}"
        echo -e "   Bootstrap Pipeline has critical issues that need attention."
    fi

    echo
    echo -e "${BLUE}📋 System Information:${NC}"
    echo -e "   • Namespace: $SYSTEM_NAMESPACE"
    echo -e "   • Kubernetes Context: $(kubectl config current-context 2>/dev/null || echo 'unknown')"
    echo -e "   • Tekton Version: $(kubectl get deployment tekton-pipelines-controller -n tekton-pipelines -o jsonpath='{.metadata.labels.app\.kubernetes\.io/version}' 2>/dev/null || echo 'unknown')"

    if [ "$FAILED_CHECKS" -gt 0 ]; then
        echo
        echo -e "${RED}🔧 Recommended Actions:${NC}"
        echo -e "   1. Review failed checks above"
        echo -e "   2. Run './install.sh' to reinstall missing components"
        echo -e "   3. Check Kubernetes cluster permissions"
        echo -e "   4. Verify Tekton Pipelines installation"
        return 1
    fi

    if [ "$WARNING_CHECKS" -gt 0 ]; then
        echo
        echo -e "${YELLOW}💡 Recommendations:${NC}"
        echo -e "   1. Review warning messages above"
        echo -e "   2. Consider adding resource quotas for better resource management"
        echo -e "   3. Monitor PipelineRun activity after deploying RepoSentry"
    fi

    return 0
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
        --no-connectivity)
            CHECK_CONNECTIVITY="false"
            shift
            ;;
        --quick)
            CHECK_CONNECTIVITY="false"
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
    echo -e "${GREEN}🔍 Validating RepoSentry Bootstrap Pipeline Installation...${NC}"
    echo -e "${BLUE}📁 Working directory: $SCRIPT_DIR${NC}"
    echo -e "${BLUE}🎯 Target namespace: $SYSTEM_NAMESPACE${NC}"
    echo

    # Run validation checks
    check_prerequisites
    validate_namespace || exit 1
    validate_pipeline
    validate_tasks
    validate_tekton_triggers
    validate_rbac
    validate_resource_limits
    test_pipeline_connectivity
    check_recent_activity
    validate_against_static_files

    # Show summary and exit with appropriate code
    show_validation_summary
    exit $?
}

# Run main function
main "$@"
