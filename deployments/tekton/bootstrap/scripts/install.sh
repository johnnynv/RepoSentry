#!/bin/bash

# RepoSentry Bootstrap Pipeline Installation Script
# This script deploys the Bootstrap Pipeline YAML files in the current directory

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

# Ingress Configuration (integrated from configure.sh)
DEFAULT_INGRESS_CLASS="nginx"
DEFAULT_WEBHOOK_HOST="webhook.127.0.0.1.nip.io"
DEFAULT_SSL_REDIRECT="false"
INGRESS_CLASS="${BOOTSTRAP_INGRESS_CLASS:-$DEFAULT_INGRESS_CLASS}"
WEBHOOK_HOST="${BOOTSTRAP_WEBHOOK_HOST:-$DEFAULT_WEBHOOK_HOST}"
SSL_REDIRECT="${BOOTSTRAP_SSL_REDIRECT:-$DEFAULT_SSL_REDIRECT}"
AUTO_CONFIGURE="${BOOTSTRAP_AUTO_CONFIGURE:-true}"

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

# Function to detect cluster configuration (integrated from configure.sh)
detect_cluster_config() {
    print_status "Auto-detecting cluster configuration..."
    
    # Detect Ingress Controller
    local ingress_classes=$(kubectl get ingressclass --no-headers 2>/dev/null | awk '{print $1}' | tr '\n' ' ')
    if [ -n "$ingress_classes" ]; then
        print_verbose "Found Ingress classes: $ingress_classes"
        
        # Priority: nginx > traefik > istio > first available
        if echo "$ingress_classes" | grep -q "nginx"; then
            INGRESS_CLASS="nginx"
        elif echo "$ingress_classes" | grep -q "traefik"; then
            INGRESS_CLASS="traefik"
        elif echo "$ingress_classes" | grep -q "istio"; then
            INGRESS_CLASS="istio"
        else
            INGRESS_CLASS=$(echo "$ingress_classes" | awk '{print $1}')
        fi
        print_success "Detected Ingress Controller: $INGRESS_CLASS"
    else
        print_warning "No Ingress Controllers detected, using default: $DEFAULT_INGRESS_CLASS"
        INGRESS_CLASS="$DEFAULT_INGRESS_CLASS"
    fi
    
    # Auto-detect webhook host if using default
    if [ "$WEBHOOK_HOST" = "$DEFAULT_WEBHOOK_HOST" ]; then
        # Try to get cluster IP or use localhost
        local cluster_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null)
        if [ -n "$cluster_ip" ]; then
            WEBHOOK_HOST="webhook.$cluster_ip.nip.io"
            print_success "Auto-configured webhook host: $WEBHOOK_HOST"
        else
            print_verbose "Using default webhook host: $WEBHOOK_HOST"
        fi
    fi
}

# Function to apply Ingress configuration (integrated from configure.sh)
apply_ingress_configuration() {
    print_status "Applying Ingress configuration..."
    
    local ingress_file="$YAML_DIR/11-ingress.yaml"
    if [ ! -f "$ingress_file" ]; then
        print_error "Ingress file not found: $ingress_file"
        return 1
    fi
    
    print_verbose "Updating Ingress configuration..."
    print_verbose "  Ingress Class: $INGRESS_CLASS"
    print_verbose "  Webhook Host: $WEBHOOK_HOST"
    print_verbose "  SSL Redirect: $SSL_REDIRECT"
    
    # Apply configuration to Ingress file
    if [ "$DRY_RUN" = "false" ]; then
        sed -i "s/ingressClassName: .*/ingressClassName: $INGRESS_CLASS/" "$ingress_file"
        sed -i "s/host: .*/host: $WEBHOOK_HOST/" "$ingress_file"
        sed -i "s/ssl-redirect: \".*\"/ssl-redirect: \"$SSL_REDIRECT\"/" "$ingress_file"
        
        # Update annotations based on ingress class
        case "$INGRESS_CLASS" in
            "traefik")
                # Convert nginx annotations to traefik
                sed -i 's/nginx\.ingress\.kubernetes\.io/traefik.ingress.kubernetes.io/g' "$ingress_file"
                ;;
            "istio")
                # Remove nginx-specific annotations for istio
                sed -i '/nginx\.ingress\.kubernetes\.io/d' "$ingress_file"
                ;;
        esac
        
        print_success "Ingress configuration applied"
    else
        print_status "DRY-RUN: Would update $ingress_file with:"
        print_status "  ingressClassName: $INGRESS_CLASS"
        print_status "  host: $WEBHOOK_HOST"
        print_status "  ssl-redirect: $SSL_REDIRECT"
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Install RepoSentry Bootstrap Pipeline infrastructure to Kubernetes cluster.
Automatically detects and configures Ingress Controller settings.

OPTIONS:
    -h, --help                     Show this help message
    -n, --namespace NAMESPACE      System namespace (default: $DEFAULT_NAMESPACE)
    -d, --dry-run                 Perform dry run without actual deployment
    -v, --verbose                 Enable verbose output
    --force                       Force deployment even if resources exist
    --ingress-class CLASS         Override detected Ingress Controller class
    --webhook-host HOST           Override auto-detected webhook host
    --ssl-redirect BOOL           Enable/disable SSL redirect (true/false)
    --no-auto-configure           Disable automatic Ingress configuration

ENVIRONMENT VARIABLES:
    BOOTSTRAP_NAMESPACE           Override system namespace
    BOOTSTRAP_DRY_RUN            Set to 'true' for dry run
    BOOTSTRAP_VERBOSE            Set to 'true' for verbose output
    BOOTSTRAP_INGRESS_CLASS      Override Ingress Controller class
    BOOTSTRAP_WEBHOOK_HOST       Override webhook host
    BOOTSTRAP_SSL_REDIRECT       Override SSL redirect setting
    BOOTSTRAP_AUTO_CONFIGURE     Set to 'false' to disable auto-configuration

EXAMPLES:
    # Basic installation (deploys to reposentry-system namespace)
    $0

    # Install to custom namespace
    $0 --namespace my-reposentry-system

    # Dry run to see what would be deployed
    $0 --dry-run

    # Verbose installation
    $0 --verbose

PREREQUISITE:
    Make sure you have:
    - kubectl installed and configured
    - Tekton Pipelines installed in your cluster
    - Cluster admin permissions for RBAC setup

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

    print_success "Prerequisites check completed"
}

# Function to verify generated files
verify_generated_files() {
    print_status "Verifying generated files in current directory..."

    local required_files=(
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

    for file in "${required_files[@]}"; do
        if [ ! -f "$YAML_DIR/$file" ]; then
            print_error "Required file not found: $YAML_DIR/$file"
            print_error "Please ensure all Bootstrap Pipeline YAML files are present"
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
        DRY_RUN_FLAG="--dry-run=server"
    else
        DRY_RUN_FLAG=""
    fi

    # Apply files in order
    local files=(
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

    for file in "${files[@]}"; do
        print_status "Applying $file..."
        kubectl apply -f "$YAML_DIR/$file" $DRY_RUN_FLAG
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
        "reposentry-bootstrap-run"
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

    # Check Tekton Triggers components
    if kubectl get triggerbinding reposentry-bootstrap-binding -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Bootstrap TriggerBinding exists"
    else
        print_error "Bootstrap TriggerBinding not found"
        exit 1
    fi

    if kubectl get triggertemplate reposentry-bootstrap-template -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Bootstrap TriggerTemplate exists"
    else
        print_error "Bootstrap TriggerTemplate not found"
        exit 1
    fi

    if kubectl get eventlistener reposentry-standard-eventlistener -n "$SYSTEM_NAMESPACE" &> /dev/null; then
        print_success "Bootstrap EventListener exists"
    else
        print_error "Bootstrap EventListener not found"
        exit 1
    fi

    print_success "Deployment verification completed successfully"
}

# Function to restart EventListener Pod to ensure configuration takes effect
restart_eventlistener_pod() {
    print_status "Restarting EventListener Pod to ensure configuration takes effect..."
    
    # Delete EventListener pods to force recreation with new configuration
    if kubectl delete pods -n "$SYSTEM_NAMESPACE" -l eventlistener=reposentry-standard-eventlistener --ignore-not-found &> /dev/null; then
        print_status "EventListener Pod deleted, waiting for recreation..."
        
        # Wait for new pod to be ready
        local max_wait=30
        local wait_count=0
        
        while [ $wait_count -lt $max_wait ]; do
            if kubectl get pods -n "$SYSTEM_NAMESPACE" -l eventlistener=reposentry-standard-eventlistener -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
                print_success "EventListener Pod restarted and ready"
                return 0
            fi
            echo -n "."
            sleep 1
            ((wait_count++))
        done
        
        print_warning "EventListener Pod restart may still be in progress"
    else
        print_warning "No EventListener Pod found to restart"
    fi
}

# Function to show post-installation information
show_post_install_info() {
    echo
    echo -e "${GREEN}🎉 Bootstrap Pipeline Installation Completed!${NC}"
    echo
    echo -e "${BLUE}📋 Summary:${NC}"
    echo "  • System Namespace: ${SYSTEM_NAMESPACE}"
    echo "  • Bootstrap Pipeline: reposentry-bootstrap-pipeline"
    echo "  • Bootstrap Tasks: 5 tasks deployed"
    echo "  • RBAC: ServiceAccount, ClusterRole, ClusterRoleBinding configured"
    echo "  • Tekton Triggers: EventListener, TriggerBinding, TriggerTemplate deployed"
    echo "  • Webhook: EventListener exposed via Ingress"
    echo
    echo -e "${BLUE}🔍 Verification Commands:${NC}"
    echo "  # Check all resources"
    echo "  kubectl get all -n ${SYSTEM_NAMESPACE}"
    echo
    echo "  # Check RBAC"
    echo "  kubectl get serviceaccount,clusterrole,clusterrolebinding | grep reposentry-bootstrap"
    echo
    echo "  # View Pipeline definition"
    echo "  kubectl describe pipeline reposentry-bootstrap-pipeline -n ${SYSTEM_NAMESPACE}"
    echo
    echo -e "${BLUE}📚 Next Steps:${NC}"
    echo "  1. Configure RepoSentry with Tekton enabled"
    echo "  2. Start RepoSentry - it will automatically use the Bootstrap Pipeline"
    echo "  3. Monitor PipelineRuns: kubectl get pipelineruns -n ${SYSTEM_NAMESPACE}"
    echo
    echo -e "${BLUE}🔧 Troubleshooting:${NC}"
    echo "  • Check logs: kubectl logs -n ${SYSTEM_NAMESPACE} -l tekton.dev/pipeline=reposentry-bootstrap-pipeline"
    echo "  • See README.md for detailed troubleshooting"
    echo
    echo -e "${BLUE}💡 Tip:${NC}"
    echo "  You can now test the Bootstrap Pipeline by triggering a CloudEvent to it."
    echo "  RepoSentry will automatically do this when it detects .tekton/ resources in monitored repositories."
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
        --ingress-class)
            INGRESS_CLASS="$2"
            shift 2
            ;;
        --webhook-host)
            WEBHOOK_HOST="$2"
            shift 2
            ;;
        --ssl-redirect)
            SSL_REDIRECT="$2"
            shift 2
            ;;
        --no-auto-configure)
            AUTO_CONFIGURE="false"
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
    echo -e "${BLUE}📁 Working directory: $SCRIPT_DIR${NC}"
    echo

    check_prerequisites
    
    # Auto-configure Ingress if enabled
    if [ "$AUTO_CONFIGURE" = "true" ]; then
        detect_cluster_config
    else
        print_status "Auto-configuration disabled, using provided settings"
        print_status "Ingress Class: $INGRESS_CLASS"
        print_status "Webhook Host: $WEBHOOK_HOST"
        print_status "SSL Redirect: $SSL_REDIRECT"
    fi
    
    verify_generated_files
    deploy_bootstrap_pipeline
    
    # Apply Ingress configuration before verification
    apply_ingress_configuration
    
    verify_deployment
    restart_eventlistener_pod
    show_post_install_info
}

# Run main function
main "$@"
