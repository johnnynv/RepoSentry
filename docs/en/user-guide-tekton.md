# RepoSentry Tekton Integration User Guide

## 🎯 Overview

RepoSentry's Tekton integration feature allows you to define Tekton pipelines in your own code repositories. When code changes occur, these pipelines will be automatically executed. This process is completely transparent to you - you just need to add a `.tekton/` directory and related YAML files to your repository.

### 🔧 Currently Available Features
- ✅ **Auto-Detection**: Monitor changes in the `.tekton/` directory of your repository
- ✅ **Transparent Execution**: Automatically execute your Tekton pipelines after code commits
- ✅ **Security Isolation**: Provide independent execution environments for your repository
- ✅ **Pre-deployed Infrastructure**: Based on stable pre-deployed Bootstrap Pipeline

### 📋 Long-term Planned Features
- 📋 **Configurable Paths**: Administrator-configurable detection paths (long-term plan)
- 📋 **Smart Discovery**: Automatically discover Tekton resources in your repository and provide recommendations (long-term plan)
- 📋 **Enterprise Governance**: Hierarchical configuration management and policy governance (long-term plan)

## 🚀 Quick Start

### Step 1: Create Tekton Resources in Your Repository

Create a `.tekton/` directory in the root of your code repository:

```bash
mkdir .tekton
cd .tekton
```

### Step 2: Create Your First Pipeline

Create a simple build and test pipeline:

```yaml
# .tekton/pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: my-app-ci
  labels:
    app: my-app
spec:
  params:
    - name: repository-url
      type: string
      description: "Git repository URL"
    - name: commit-sha
      type: string
      description: "Git commit SHA"
    - name: repository-name
      type: string
      description: "Repository name"
  
  workspaces:
    - name: source-code
    - name: docker-credentials
      optional: true
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    - name: run-tests
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: node:16
            workingDir: $(workspaces.source.path)
            script: |
              #!/bin/bash
              echo "🧪 Running tests for $(params.repository-name)..."
              
              # Check if package.json exists
              if [ -f "package.json" ]; then
                npm install
                npm test
              fi
              
              # Check if go.mod exists
              if [ -f "go.mod" ]; then
                go test ./...
              fi
              
              # Check if pom.xml exists
              if [ -f "pom.xml" ]; then
                mvn test
              fi
              
              echo "✅ Tests completed!"
      workspaces:
        - name: source
          workspace: source-code
    
    - name: build-image
      runAfter: ["run-tests"]
      taskSpec:
        workspaces:
          - name: source
          - name: dockerconfig
            optional: true
        params:
          - name: image-name
            default: "$(params.repository-name):$(params.commit-sha)"
        steps:
          - name: build
            image: gcr.io/kaniko-project/executor:latest
            workingDir: $(workspaces.source.path)
            script: |
              #!/busybox/sh
              echo "🔨 Building container image..."
              
              # Check if Dockerfile exists
              if [ -f "Dockerfile" ]; then
                echo "Found Dockerfile, building image: $(params.image-name)"
                /kaniko/executor \
                  --context $(workspaces.source.path) \
                  --dockerfile $(workspaces.source.path)/Dockerfile \
                  --destination $(params.image-name) \
                  --no-push
              else
                echo "⚠️  No Dockerfile found, skipping image build"
              fi
            env:
              - name: DOCKER_CONFIG
                value: $(workspaces.dockerconfig.path)
      workspaces:
        - name: source
          workspace: source-code
        - name: dockerconfig
          workspace: docker-credentials
```

### Step 3: Commit Your Code

Commit your `.tekton/` directory to the Git repository:

```bash
git add .tekton/
git commit -m "Add Tekton CI pipeline"
git push origin main
```

### Step 4: Observe Execution Results

After committing your code, RepoSentry will automatically detect your Tekton resources and execute the Pipeline. You can view the execution status through the following methods:

```bash
# View PipelineRuns in your namespace (using hash namespace)
kubectl get pipelineruns -n reposentry-user-repo-{namespace-hash}

# View Pipeline execution logs
kubectl logs -f pipelinerun/{pipelinerun-name} -n reposentry-user-repo-{namespace-hash}

# Note: namespace-hash is a hash value generated based on your repository information
# You can query your namespace with the following command:
kubectl get namespaces -l reposentry.dev/repository={your-repo}

# Namespace example: reposentry-user-repo-abc123def456
```

## 📁 Recommended Directory Structure

Recommended `.tekton/` directory structure:

```
.tekton/
├── pipeline.yaml              # Main pipeline definition
├── tasks/                     # Custom tasks
│   ├── build-task.yaml
│   ├── test-task.yaml
│   └── deploy-task.yaml
├── pipelines/                 # Multiple pipelines
│   ├── ci-pipeline.yaml
│   ├── cd-pipeline.yaml
│   └── release-pipeline.yaml
├── triggers/                  # Trigger configurations (optional)
│   ├── binding.yaml
│   └── template.yaml
├── configs/                   # Configuration files
│   ├── workspace-template.yaml
│   └── secrets-template.yaml
└── environments/              # Environment-specific configurations
    ├── dev/
    │   └── pipeline.yaml
    ├── staging/
    │   └── pipeline.yaml
    └── prod/
        └── pipeline.yaml
```

**Note**:
- ✅ Supports creating arbitrary levels of subdirectories under `.tekton/`
- ✅ All `.yaml` and `.yml` files will be automatically detected
- ✅ Can organize file structure by function, environment, or team
- ❌ Does not support Tekton resources outside the `.tekton/` directory

## 🔧 Common Tekton Resource Examples

### Custom Task Example

```yaml
# .tekton/tasks/build-task.yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: custom-build
spec:
  params:
    - name: project-type
      type: string
      default: "nodejs"
    - name: build-args
      type: string
      default: ""
  
  workspaces:
    - name: source
  
  steps:
    - name: detect-project-type
      image: alpine
      script: |
        #!/bin/sh
        cd $(workspaces.source.path)
        
        if [ -f "package.json" ]; then
          echo "nodejs" > /tmp/project-type
        elif [ -f "go.mod" ]; then
          echo "golang" > /tmp/project-type
        elif [ -f "pom.xml" ]; then
          echo "java" > /tmp/project-type
        elif [ -f "requirements.txt" ]; then
          echo "python" > /tmp/project-type
        else
          echo "unknown" > /tmp/project-type
        fi
    
    - name: build-project
      image: alpine
      script: |
        #!/bin/sh
        PROJECT_TYPE=$(cat /tmp/project-type)
        cd $(workspaces.source.path)
        
        echo "🔨 Building $PROJECT_TYPE project..."
        
        case $PROJECT_TYPE in
          "nodejs")
            npm install
            npm run build $(params.build-args)
            ;;
          "golang")
            go build $(params.build-args) ./...
            ;;
          "java")
            mvn compile $(params.build-args)
            ;;
          "python")
            pip install -r requirements.txt
            python setup.py build $(params.build-args)
            ;;
          *)
            echo "⚠️  Unknown project type, skipping build"
            ;;
        esac
        
        echo "✅ Build completed!"
```

### Multi-Environment Deployment Pipeline

```yaml
# .tekton/pipeline-deploy.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: my-app-deploy
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: repository-name
    - name: target-environment
      default: "development"
  
  workspaces:
    - name: source-code
    - name: docker-credentials
  
  tasks:
    - name: clone-source
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    - name: build-and-push
      runAfter: ["clone-source"]
      taskSpec:
        workspaces:
          - name: source
          - name: dockerconfig
        params:
          - name: image-name
            default: "my-registry/$(params.repository-name):$(params.commit-sha)"
        steps:
          - name: build-and-push
            image: gcr.io/kaniko-project/executor:latest
            script: |
              #!/busybox/sh
              /kaniko/executor \
                --context $(workspaces.source.path) \
                --dockerfile $(workspaces.source.path)/Dockerfile \
                --destination $(params.image-name)
            env:
              - name: DOCKER_CONFIG
                value: $(workspaces.dockerconfig.path)
      workspaces:
        - name: source
          workspace: source-code
        - name: dockerconfig
          workspace: docker-credentials
    
    - name: deploy-to-environment
      runAfter: ["build-and-push"]
      taskSpec:
        params:
          - name: environment
          - name: image
          - name: app-name
        steps:
          - name: deploy
            image: bitnami/kubectl
            script: |
              #!/bin/bash
              echo "🚀 Deploying to $(params.environment) environment..."
              
              # Select namespace based on environment
              case $(params.environment) in
                "development")
                  NAMESPACE="dev-$(params.app-name)"
                  ;;
                "staging")
                  NAMESPACE="staging-$(params.app-name)"
                  ;;
                "production")
                  NAMESPACE="prod-$(params.app-name)"
                  ;;
                *)
                  echo "❌ Unknown environment: $(params.environment)"
                  exit 1
                  ;;
              esac
              
              # Create namespace (if it doesn't exist)
              kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
              
              # Deploy application
              cat <<EOF | kubectl apply -f -
              apiVersion: apps/v1
              kind: Deployment
              metadata:
                name: $(params.app-name)
                namespace: $NAMESPACE
              spec:
                replicas: 1
                selector:
                  matchLabels:
                    app: $(params.app-name)
                template:
                  metadata:
                    labels:
                      app: $(params.app-name)
                  spec:
                    containers:
                    - name: app
                      image: $(params.image)
                      ports:
                      - containerPort: 8080
              EOF
              
              echo "✅ Deployment completed!"
      params:
        - name: environment
          value: $(params.target-environment)
        - name: image
          value: "my-registry/$(params.repository-name):$(params.commit-sha)"
        - name: app-name
          value: $(params.repository-name)
```

## 🔧 Advanced Configuration

### Conditional Execution

Execute different tasks based on branches or file changes:

```yaml
# .tekton/conditional-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: conditional-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: branch-name
  
  workspaces:
    - name: source-code
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    # Only run deployment on main branch
    - name: deploy-to-production
      when:
        - input: "$(params.branch-name)"
          operator: in
          values: ["main", "master"]
      runAfter: ["git-clone"]
      taskSpec:
        steps:
          - name: deploy
            image: alpine
            script: |
              echo "🚀 Deploying to production (branch: $(params.branch-name))..."
              # Deployment logic...
    
    # Only run tests on non-main branches
    - name: run-dev-tests
      when:
        - input: "$(params.branch-name)"
          operator: notin
          values: ["main", "master"]
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: alpine
            script: |
              echo "🧪 Running development tests (branch: $(params.branch-name))..."
              # Testing logic...
      workspaces:
        - name: source
          workspace: source-code
```

### Parallel Task Execution

```yaml
# .tekton/parallel-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: parallel-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
  
  workspaces:
    - name: source-code
  
  tasks:
    - name: git-clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      params:
        - name: url
          value: $(params.repository-url)
        - name: revision
          value: $(params.commit-sha)
      workspaces:
        - name: output
          workspace: source-code
    
    # Parallel execution tasks
    - name: lint-code
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: lint
            image: alpine
            script: |
              echo "🔍 Running code linting..."
              # Linting logic...
      workspaces:
        - name: source
          workspace: source-code
    
    - name: security-scan
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: scan
            image: alpine
            script: |
              echo "🔒 Running security scan..."
              # Security scanning logic...
      workspaces:
        - name: source
          workspace: source-code
    
    - name: unit-tests
      runAfter: ["git-clone"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: test
            image: alpine
            script: |
              echo "🧪 Running unit tests..."
              # Unit testing logic...
      workspaces:
        - name: source
          workspace: source-code
    
    # Execute after all parallel tasks complete
    - name: build-application
      runAfter: ["lint-code", "security-scan", "unit-tests"]
      taskSpec:
        workspaces:
          - name: source
        steps:
          - name: build
            image: alpine
            script: |
              echo "🔨 Building application..."
              # Build logic...
      workspaces:
        - name: source
          workspace: source-code
```

## 🔍 Debugging and Troubleshooting

### View Execution Logs

```bash
# List all PipelineRuns in your namespace
kubectl get pipelineruns -n reposentry-user-repo-{namespace-hash}

# View detailed information for a specific PipelineRun
kubectl describe pipelinerun {pipelinerun-name} -n reposentry-user-repo-{namespace-hash}

# View real-time logs
kubectl logs -f pipelinerun/{pipelinerun-name} -n reposentry-user-repo-{namespace-hash}

# View logs for a specific task
kubectl logs -f pipelinerun/{pipelinerun-name} -c step-{step-name} -n reposentry-user-repo-{namespace-hash}
```

### Common Issue Resolution

#### 1. Pipeline Not Auto-triggering

**Possible Causes**:
- `.tekton/` directory doesn't exist or is empty
- YAML file format errors
- RepoSentry didn't detect changes

**Solutions**:
```bash
# Check .tekton directory structure
ls -la .tekton/

# Validate YAML file format
yamllint .tekton/*.yaml

# Manually trigger detection (if you have permissions)
curl -X POST http://reposentry-api/api/v1/repositories/{repo}/trigger
```

#### 2. Task Execution Failures

**Common Errors**:
```yaml
# Incorrect image reference
steps:
  - name: build
    image: node:16-invalid  # Image doesn't exist
    
# Incorrect working directory
steps:
  - name: test
    workingDir: /nonexistent/path  # Path doesn't exist
    
# Insufficient permissions
steps:
  - name: deploy
    script: |
      kubectl apply -f deployment.yaml  # May not have permissions
```

**Solutions**:
- Use valid image tags
- Ensure working directories exist
- Check required permissions and RBAC configuration

#### 3. Resource Quota Exceeded

**Error Message**:
```
Error: pods "my-task-pod" is forbidden: exceeded quota
```

**Solutions**:
- Reduce number of parallel tasks
- Optimize resource requests and limits
- Contact administrator to adjust quotas

## 📚 Best Practices

### 1. Resource Optimization

```yaml
# Set appropriate resource limits for tasks
taskSpec:
  stepTemplate:
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "200m"
  steps:
    - name: build
      # ... other configuration
```

### 2. Image Selection

```yaml
# Use lightweight images
steps:
  - name: test
    image: alpine:3.18  # instead of ubuntu:latest
    
  # Use specific version tags
  - name: build
    image: node:16.20.0-alpine  # instead of node:latest
```

### 3. Security Practices

```yaml
# Don't hardcode sensitive information in YAML
steps:
  - name: deploy
    env:
      - name: API_KEY
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: api-key
    script: |
      # Use environment variables
      curl -H "Authorization: Bearer $API_KEY" ...
```

### 4. Error Handling

```yaml
steps:
  - name: robust-task
    image: alpine
    script: |
      #!/bin/bash
      set -euo pipefail  # Strict error handling
      
      # Check for required files
      if [ ! -f "required-file.txt" ]; then
        echo "❌ Required file not found"
        exit 1
      fi
      
      # Execute operation and check result
      if ! some-command; then
        echo "❌ Command failed"
        exit 1
      fi
      
      echo "✅ Task completed successfully"
```

### 5. Reusability

```yaml
# Use parameters to make Pipeline more flexible
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: reusable-ci
spec:
  params:
    - name: repository-url
    - name: commit-sha
    - name: build-image
      default: "node:16"
    - name: test-command
      default: "npm test"
    - name: build-command
      default: "npm run build"
  
  tasks:
    - name: flexible-build
      taskSpec:
        params:
          - name: build-image
          - name: test-cmd
          - name: build-cmd
        steps:
          - name: test
            image: $(params.build-image)
            script: $(params.test-cmd)
          - name: build
            image: $(params.build-image)
            script: $(params.build-cmd)
      params:
        - name: build-image
          value: $(params.build-image)
        - name: test-cmd
          value: $(params.test-command)
        - name: build-cmd
          value: $(params.build-command)
```

## 🔗 Related Resources

- [Tekton Pipelines Official Documentation](https://tekton.dev/docs/pipelines/)
- [Tekton Tasks Catalog](https://hub.tekton.dev/)
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [YAML Syntax Guide](https://yaml.org/spec/)

## 💬 Getting Help

If you encounter issues during use:

1. **Check Logs**: First examine the PipelineRun execution logs
2. **Validate YAML**: Ensure your Tekton resource format is correct
3. **Check Permissions**: Confirm your Pipeline has sufficient permissions to perform required operations
4. **Reference Examples**: Review the examples and best practices in this guide
5. **Contact Support**: If issues persist, please contact your platform administrator

---

**Note**: RepoSentry's Tekton integration feature is completely transparent. You don't need to configure any Webhooks or perform additional setup. Simply add a `.tekton/` directory and related YAML files to your repository, and the system will automatically detect and execute your Pipeline.
