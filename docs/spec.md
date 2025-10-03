# k8s-manifests-lint: Technical Specification

## Overview

A pluggable linter for Kubernetes manifests inspired by golangci-lint, providing a unified interface for running multiple linters against Kubernetes resources rendered from various sources (Kustomize, Helm, Go templates).

## Core Requirements

### Technology Stack
- **Language**: Go 1.24+
- **CLI Framework**: [cobra](https://github.com/spf13/cobra)
- **Configuration**: [viper](https://github.com/spf13/viper)
- **Manifest Rendering**: [k8s-manifests-lib](https://github.com/lburgazzoli/k8s-manifests-lib)
- **K8s API**: k8s.io/api, k8s.io/apimachinery

### Key Features
- Multiple pre-defined linters (enable/disable individually)
- Per-linter configuration
- Support for Kustomize, Helm, and Go templates
- YAML-based configuration file
- GitHub Action integration
- Structured output (JSON, YAML, text)
- Exit codes for CI/CD integration

## Architecture

### Project Structure
```
k8s-manifests-lint/
├── cmd/
│   └── k8s-manifests-lint/
│       └── main.go
├── pkg/
│   ├── config/
│   │   ├── config.go          # Configuration structures
│   │   └── loader.go           # Config file loading
│   ├── renderer/
│   │   ├── renderer.go         # Interface for manifest rendering
│   │   ├── kustomize.go        # Kustomize renderer
│   │   ├── helm.go             # Helm renderer
│   │   └── template.go         # Go template renderer
│   ├── linter/
│   │   ├── linter.go           # Linter interface
│   │   ├── registry.go         # Linter registration
│   │   ├── runner.go           # Linter execution engine
│   │   └── result.go           # Result types
│   ├── linters/
│   │   ├── resource_limits.go  # Check resource limits
│   │   ├── security.go         # Security best practices
│   │   ├── labels.go           # Required labels
│   │   ├── replicas.go         # Replica count checks
│   │   ├── probe.go            # Health/readiness probes
│   │   ├── image.go            # Image tag checks
│   │   ├── rbac.go             # ClusterRoleBinding security
│   │   └── ...                 # Additional linters
│   ├── output/
│   │   ├── formatter.go        # Output formatting
│   │   ├── text.go             # Text output
│   │   ├── json.go             # JSON output
│   │   └── github.go           # GitHub Actions format
│   └── k8s/
│       └── objects.go          # K8s object utilities
├── action.yml                  # GitHub Action definition (composite)
├── .k8s-manifests-lint.yaml    # Example config
├── go.mod
├── go.sum
└── README.md
```

## Configuration File Format

### .k8s-manifests-lint.yaml

```yaml
# Source configuration
sources:
  - type: kustomize
    path: ./overlays/production
  - type: helm
    chart: ./charts/myapp
    values: ./values.yaml
  - type: template
    path: ./templates/*.yaml
    data: ./data.yaml

# Linters configuration
linters:
  enable:
    - resource-limits
    - security-context
    - required-labels
    - replica-count
    - health-probes
    - image-tags
    - pod-disruption-budget
    - cluster-role-binding-security
  
  disable:
    - deprecated-api

  # Per-linter settings
  settings:
    resource-limits:
      require-cpu-limit: true
      require-memory-limit: true
      require-cpu-request: true
      require-memory-request: true
      exclude-namespaces:
        - kube-system
    
    required-labels:
      labels:
        - app
        - version
        - environment
      exclude-kinds:
        - ConfigMap
        - Secret
    
    security-context:
      require-run-as-non-root: true
      require-read-only-root-filesystem: false
      disallow-privilege-escalation: true
    
    replica-count:
      min-replicas: 2
      exclude-kinds:
        - DaemonSet
    
    image-tags:
      disallow-latest: true
      require-digest: false
      allowed-registries:
        - docker.io
        - gcr.io
        - ghcr.io
    
    cluster-role-binding-security:
      disallowed-groups:
        - system:authenticated
        - system:unauthenticated
        - system:serviceaccounts
      warn-namespace-groups: true
      allowed-roles-for-broad-groups:
        - system:basic-user
        - system:discovery

# Output configuration
output:
  format: text  # text, json, yaml, github-actions
  show-source: true
  color: auto

# Exclusions
exclude:
  # Exclude specific resources
  resources:
    - kind: ConfigMap
      name: kube-root-ca.crt
      namespace: "*"
  
  # Exclude paths
  paths:
    - "*/test/*"
    - "*/examples/*"

# Run configuration
run:
  concurrency: 4
  timeout: 5m
  skip-dirs:
    - vendor
    - .git
```

## Core Interfaces

### Linter Interface

```go
package linter

import (
    "context"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Linter defines the interface for all linters
type Linter interface {
    // Name returns the linter name
    Name() string
    
    // Description returns what the linter checks
    Description() string
    
    // Lint runs the linter against a Kubernetes object
    Lint(ctx context.Context, obj *unstructured.Unstructured) ([]Issue, error)
    
    // Configure sets up the linter with settings from config
    Configure(settings map[string]interface{}) error
}

// Issue represents a linting issue
type Issue struct {
    Severity    Severity
    Linter      string
    Message     string
    Resource    ResourceRef
    Field       string    // JSONPath to the problematic field
    Suggestion  string    // Optional fix suggestion
}

type Severity string

const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
    SeverityInfo    Severity = "info"
)

type ResourceRef struct {
    APIVersion string
    Kind       string
    Namespace  string
    Name       string
}
```

### Renderer Interface

```go
package renderer

import (
    "context"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Renderer defines the interface for manifest rendering
type Renderer interface {
    // Render produces Kubernetes objects from source
    Render(ctx context.Context, source Source) ([]*unstructured.Unstructured, error)
}

// Source defines a manifest source
type Source struct {
    Type   SourceType
    Path   string
    Values map[string]interface{}
}

type SourceType string

const (
    SourceTypeKustomize SourceType = "kustomize"
    SourceTypeHelm      SourceType = "helm"
    SourceTypeTemplate  SourceType = "template"
)
```

## CLI Commands

### Installation & Execution

The tool is designed to be run without installation using `go run`:

```bash
# Run latest version
go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run

# Run specific version
go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@v1.0.0 run

# Or install locally
go install github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest
k8s-manifests-lint run
```

### Main Command
```bash
k8s-manifests-lint [flags]
# or
go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest [flags]
```

**Flags:**
- `--config, -c`: Config file path (default: .k8s-manifests-lint.yaml)
- `--enable-linter`: Enable specific linter(s)
- `--disable-linter`: Disable specific linter(s)
- `--format`: Output format (text|json|yaml|github-actions)
- `--no-color`: Disable colored output
- `--fail-on-warning`: Exit with error on warnings

### Subcommands

#### linters
List all available linters with descriptions
```bash
k8s-manifests-lint linters [--enabled-only]
```

#### run
Explicitly run linting (default command)
```bash
k8s-manifests-lint run [flags] [path...]
```

#### config
Validate or generate config file
```bash
k8s-manifests-lint config validate
k8s-manifests-lint config init
```

#### version
Show version information
```bash
k8s-manifests-lint version
```

## Pre-defined Linters

### 1. resource-limits
Ensures containers have resource requests and limits defined.

**Checks:**
- CPU requests/limits presence
- Memory requests/limits presence
- Reasonable ratios between requests and limits

**Configuration:**
```yaml
resource-limits:
  require-cpu-limit: true
  require-memory-limit: true
  require-cpu-request: true
  require-memory-request: true
  max-cpu-limit: "4"
  max-memory-limit: "8Gi"
```

### 2. security-context
Validates pod and container security contexts.

**Checks:**
- runAsNonRoot is set
- readOnlyRootFilesystem where applicable
- allowPrivilegeEscalation is false
- Capabilities are dropped

**Configuration:**
```yaml
security-context:
  require-run-as-non-root: true
  require-read-only-root-filesystem: false
  disallow-privilege-escalation: true
  required-dropped-capabilities:
    - ALL
```

### 3. required-labels
Ensures resources have required labels.

**Configuration:**
```yaml
required-labels:
  labels:
    - app
    - version
    - environment
    - owner
  exclude-kinds:
    - Secret
    - ConfigMap
```

### 4. replica-count
Checks deployment replica counts for HA.

**Configuration:**
```yaml
replica-count:
  min-replicas: 2
  exclude-kinds:
    - DaemonSet
  exclude-namespaces:
    - dev
```

### 5. health-probes
Ensures pods have liveness and readiness probes.

**Configuration:**
```yaml
health-probes:
  require-liveness: true
  require-readiness: true
  exclude-kinds:
    - Job
    - CronJob
```

### 6. image-tags
Validates container image tags.

**Checks:**
- No "latest" tag
- Specific version tags
- Allowed registries
- Optional: require digest

**Configuration:**
```yaml
image-tags:
  disallow-latest: true
  require-digest: false
  allowed-registries:
    - docker.io
    - gcr.io
  require-version-pattern: "^v?\\d+\\.\\d+\\.\\d+$"
```

### 7. pod-disruption-budget
Ensures PDBs exist for deployments.

**Configuration:**
```yaml
pod-disruption-budget:
  min-replicas-threshold: 2
  exclude-namespaces:
    - kube-system
```

### 8. deprecated-api
Warns about deprecated Kubernetes APIs.

**Configuration:**
```yaml
deprecated-api:
  target-version: "1.29"
```

### 9. namespace-isolation
Checks for proper namespace configuration.

**Checks:**
- NetworkPolicies exist
- ResourceQuotas defined
- LimitRanges configured

### 10. service-monitor
Validates ServiceMonitor for Prometheus (if using Prometheus Operator).

### 11. cluster-role-binding-security
Validates ClusterRoleBindings for overly permissive group assignments.

**Checks:**
- Detects bindings to broad system groups
- Warns about potential security risks
- Ensures principle of least privilege

**Dangerous Groups:**
- `system:authenticated` - All authenticated users (including all service accounts)
- `system:unauthenticated` - All unauthenticated users
- `system:serviceaccounts` - All service accounts across all namespaces
- Optionally: `system:serviceaccounts:<namespace>` - All SAs in a namespace

**Configuration:**
```yaml
cluster-role-binding-security:
  # Groups that should never be used in ClusterRoleBindings
  disallowed-groups:
    - system:authenticated
    - system:unauthenticated
    - system:serviceaccounts
  
  # Warn about namespace-wide service account groups
  warn-namespace-groups: true  # system:serviceaccounts:*
  
  # Allowed exceptions (by ClusterRoleBinding name)
  exceptions:
    - name: system:basic-user
      reason: "Required for basic user access"
  
  # Minimum privilege roles that are allowed with broad groups
  allowed-roles-for-broad-groups:
    - system:basic-user
    - system:discovery
    - system:public-info-viewer
  
  # Fail on critical roles with broad groups
  critical-roles:
    - cluster-admin
    - admin
    - edit
```

**Example Issues:**
```yaml
# BAD: Grants cluster-admin to all authenticated users
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: dangerous-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:authenticated  # ❌ CRITICAL: All authenticated users get cluster-admin

# BAD: Even with less privileged roles, broad groups are risky
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: risky-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:serviceaccounts  # ⚠️ WARNING: All service accounts get view access

# GOOD: Specific service account binding
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: safe-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
- kind: ServiceAccount
  name: monitoring
  namespace: observability  # ✅ Specific service account
```

## GitHub Action

### action.yml
```yaml
name: 'K8s Manifests Lint'
description: 'Lint Kubernetes manifests using k8s-manifests-lint'
author: 'Your Name'

branding:
  icon: 'check-circle'
  color: 'blue'

inputs:
  version:
    description: 'Version of k8s-manifests-lint to use (e.g., @latest, @v1.0.0)'
    required: false
    default: '@latest'
  
  config:
    description: 'Path to config file'
    required: false
    default: '.k8s-manifests-lint.yaml'
  
  fail-on-warning:
    description: 'Fail on warnings'
    required: false
    default: 'false'
  
  format:
    description: 'Output format (text|json|github-actions)'
    required: false
    default: 'github-actions'
  
  enable-linters:
    description: 'Comma-separated list of linters to enable'
    required: false
  
  disable-linters:
    description: 'Comma-separated list of linters to disable'
    required: false
  
  go-version:
    description: 'Go version to use'
    required: false
    default: '1.24'
  
  working-directory:
    description: 'Working directory to run the linter in'
    required: false
    default: '.'

outputs:
  issues-count:
    description: 'Number of issues found'
  
  errors-count:
    description: 'Number of errors'
  
  warnings-count:
    description: 'Number of warnings'

runs:
  using: 'composite'
  steps:
    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ inputs.go-version }}
        cache: false
    
    - name: Run k8s-manifests-lint
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: |
        # Build arguments array
        ARGS="run"
        
        if [ -n "${{ inputs.config }}" ]; then
          ARGS="$ARGS --config=${{ inputs.config }}"
        fi
        
        if [ -n "${{ inputs.format }}" ]; then
          ARGS="$ARGS --format=${{ inputs.format }}"
        fi
        
        if [ "${{ inputs.fail-on-warning }}" = "true" ]; then
          ARGS="$ARGS --fail-on-warning"
        fi
        
        if [ -n "${{ inputs.enable-linters }}" ]; then
          IFS=',' read -ra LINTERS <<< "${{ inputs.enable-linters }}"
          for linter in "${LINTERS[@]}"; do
            ARGS="$ARGS --enable-linter=$linter"
          done
        fi
        
        if [ -n "${{ inputs.disable-linters }}" ]; then
          IFS=',' read -ra LINTERS <<< "${{ inputs.disable-linters }}"
          for linter in "${LINTERS[@]}"; do
            ARGS="$ARGS --disable-linter=$linter"
          done
        fi
        
        # Run the linter using go run
        echo "Running: go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint${{ inputs.version }} $ARGS"
        go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint${{ inputs.version }} $ARGS
```

### Usage Example
```yaml
name: Lint K8s Manifests

on:
  pull_request:
    paths:
      - 'k8s/**'
      - '.k8s-manifests-lint.yaml'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Lint K8s Manifests
        uses: your-org/k8s-manifests-lint@v1
        with:
          version: '@v1.0.0'  # or @latest
          config: .k8s-manifests-lint.yaml
          fail-on-warning: true
          format: github-actions
          go-version: '1.24'

      # Alternative: Run directly without the action
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Run linter directly
        run: |
          go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run \
            --config=.k8s-manifests-lint.yaml \
            --format=github-actions \
            --fail-on-warning
```

### Additional Usage Examples

```yaml
# Run on multiple directories
- name: Lint multiple environments
  run: |
    for env in dev staging prod; do
      echo "Linting $env environment..."
      go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run \
        --config=.k8s-manifests-lint.yaml \
        k8s/overlays/$env/
    done

# Use in pre-commit hook
- name: Pre-commit lint
  run: |
    go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run \
      --fail-on-warning \
      --format=text

# Matrix strategy for different K8s versions
jobs:
  lint:
    strategy:
      matrix:
        k8s-version: ['1.27', '1.28', '1.29']
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Lint for K8s ${{ matrix.k8s-version }}
        run: |
          go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run \
            --enable-linter=deprecated-api
```

## Implementation Phases

### Phase 1: Core Framework
1. Set up project structure with Go modules
2. Implement cobra CLI with basic commands
3. Implement viper configuration loading
4. Create linter interface and registry
5. Implement basic output formatters

### Phase 2: Rendering Engine
1. Integrate k8s-manifests-lib
2. Implement Kustomize renderer
3. Implement Helm renderer
4. Implement Go template renderer
5. Add source configuration parsing

### Phase 3: Core Linters
1. Implement 5-6 essential linters:
    - resource-limits
    - security-context
    - required-labels
    - health-probes
    - image-tags
    - cluster-role-binding-security

### Phase 4: Advanced Features
1. Add remaining linters
2. Implement exclusion rules
3. Add concurrency support
4. Implement GitHub Actions formatter
5. Add comprehensive error handling

### Phase 5: GitHub Action
1. Create composite action.yml
2. Implement shell script for argument parsing
3. Add version input parameter support
4. Test action locally with act or similar
5. Document usage patterns (both action and direct go run)

### Phase 6: Testing & Documentation
1. Unit tests for all linters
2. Integration tests
3. Example configurations
4. README and documentation
5. Contributing guidelines

## Testing Strategy

### Local Development
During development, the tool can be run directly from source:

```bash
# Run from source
go run ./cmd/k8s-manifests-lint run --config=.k8s-manifests-lint.yaml

# Run with specific linters
go run ./cmd/k8s-manifests-lint run \
  --enable-linter=resource-limits \
  --enable-linter=security-context

# Test against example manifests
go run ./cmd/k8s-manifests-lint run ./examples/
```

### Unit Tests
- Each linter tested independently
- Mock Kubernetes objects
- Test all configuration options
- Test edge cases

### Integration Tests
- Full pipeline tests with real manifests
- Test all renderers (Kustomize, Helm, templates)
- Test configuration loading
- Test output formats

### Example Test Structure
```go
func TestResourceLimitsLinter(t *testing.T) {
    tests := []struct{
        name     string
        manifest string
        config   map[string]interface{}
        wantIssues int
    }{
        {
            name: "missing cpu limit",
            manifest: `...`,
            config: map[string]interface{}{
                "require-cpu-limit": true,
            },
            wantIssues: 1,
        },
    }
    // ... test implementation
}
```

## Exit Codes

- `0`: Success, no issues found
- `1`: Linting issues found (errors)
- `2`: Configuration error
- `3`: Runtime error
- `4`: Linting issues found (warnings, if --fail-on-warning)

## Dependencies

### go.mod
```go
module github.com/lburgazzoli/k8s-manifests-lint

go 1.24

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/lburgazzoli/k8s-manifests-lib v0.x.x
    k8s.io/api v0.29.0
    k8s.io/apimachinery v0.29.0
    k8s.io/client-go v0.29.0
    sigs.k8s.io/kustomize/api v0.15.0
    helm.sh/helm/v3 v3.13.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## Success Criteria

- All 11 linters implemented and working
- Support for Kustomize, Helm, and Go templates
- YAML configuration with per-linter settings
- Multiple output formats
- Works via `go run` without requiring installation
- GitHub Action (composite) published and tested
- Comprehensive documentation with usage examples
- >80% test coverage
- Clean, maintainable code following Go best practices
- go.mod configured for use as a library or tool

## Future Enhancements

- Plugin system for custom linters
- Auto-fix capabilities
- IDE integrations (VS Code extension)
- Pre-commit hooks
- Web UI for viewing results
- Historical tracking of issues
- Custom rule definitions via config
- Integration with policy engines (OPA, Kyverno)