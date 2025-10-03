# k8s-manifests-lint

A pluggable linter for Kubernetes manifests inspired by golangci-lint, providing a unified interface for running multiple linters against Kubernetes resources.

## Features

- **6 Pre-defined Linters**: Resource limits, security contexts, required labels, health probes, image tags, and RBAC security
- **Flexible Configuration**: YAML-based configuration with per-linter settings
- **Multiple Output Formats**: Text (colored), JSON, YAML, GitHub Actions
- **Easy to Run**: Use via `go run` without installation
- **GitHub Action**: Ready-to-use composite action for CI/CD
- **gojq Integration**: Elegant jq-style queries for writing custom linters

## Quick Start

### Run without installation

```bash
# Run on current directory
go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run

# Run on specific files
go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run path/to/manifests/
```

### Install locally

```bash
go install github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest
k8s-manifests-lint run
```

## Available Linters

| Linter | Description |
|--------|-------------|
| `resource-limits` | Ensures containers have resource requests and limits defined |
| `security-context` | Validates pod and container security contexts |
| `required-labels` | Ensures resources have required labels |
| `health-probes` | Ensures pods have liveness and readiness probes |
| `image-tags` | Validates container image tags (no latest, specific versions) |
| `cluster-role-binding-security` | Validates ClusterRoleBindings for overly permissive group assignments |

List all linters:

```bash
k8s-manifests-lint linters
```

## Configuration

Create a `.k8s-manifests-lint.yaml` file in your project root:

```yaml
linters:
  enable:
    - resource-limits
    - security-context
    - required-labels
    - health-probes
    - image-tags
    - cluster-role-binding-security

  settings:
    resource-limits:
      require-cpu-limit: true
      require-memory-limit: true
      exclude-namespaces:
        - kube-system

    required-labels:
      labels:
        - app
        - version
      exclude-kinds:
        - ConfigMap
        - Secret

    security-context:
      require-run-as-non-root: true
      disallow-privilege-escalation: true

    image-tags:
      disallow-latest: true
      allowed-registries:
        - docker.io
        - gcr.io

    cluster-role-binding-security:
      disallowed-groups:
        - system:authenticated
        - system:serviceaccounts

output:
  format: text
  color: auto

run:
  concurrency: 4
```

## Usage

### Basic Commands

```bash
# Run linters with default config
k8s-manifests-lint run

# Specify config file
k8s-manifests-lint run --config=custom-config.yaml

# Run with specific output format
k8s-manifests-lint run --format=json

# Enable/disable specific linters
k8s-manifests-lint run --enable-linter=resource-limits --disable-linter=image-tags

# Fail on warnings
k8s-manifests-lint run --fail-on-warning
```

### GitHub Actions

Use the composite action in your workflow:

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

      - name: Lint manifests
        uses: lburgazzoli/k8s-manifests-lint@v1
        with:
          config: .k8s-manifests-lint.yaml
          format: github-actions
          fail-on-warning: true
```

Or run directly:

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.24'

- name: Lint manifests
  run: |
    go run github.com/lburgazzoli/k8s-manifests-lint/cmd/k8s-manifests-lint@latest run \
      --format=github-actions \
      --fail-on-warning
```

## Examples

### Bad Deployment (9 issues)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:latest  # ❌ Using latest tag
        # ❌ No resource limits
        # ❌ No security context
        # ❌ No health probes
```

Output:

```
[error] default/Deployment/nginx-deployment: Container "nginx" uses 'latest' tag (image-tags)
  Field: spec.template.spec.containers[0].image
  Suggestion: Specify an explicit version tag

[error] default/Deployment/nginx-deployment: Container "nginx" has no resource requirements (resource-limits)
  Field: spec.template.spec.containers[0].resources
  Suggestion: Add resources.requests and resources.limits

[error] default/Deployment/nginx-deployment: Container "nginx" must set runAsNonRoot to true (security-context)
  Field: spec.template.spec.containers[0].securityContext.runAsNonRoot
  Suggestion: Add: securityContext.runAsNonRoot: true
...
```

### Good Deployment (0 issues)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
    version: "1.0.0"
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
        version: "1.0.0"
    spec:
      containers:
      - name: nginx
        image: docker.io/nginx:1.25.0  # ✅ Specific version
        resources:  # ✅ Resource limits
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "256Mi"
        securityContext:  # ✅ Security context
          runAsNonRoot: true
          allowPrivilegeEscalation: false
          capabilities:
            drop: [ALL]
        livenessProbe:  # ✅ Health probes
          httpGet:
            path: /
            port: 80
        readinessProbe:
          httpGet:
            path: /
            port: 80
```

## Exit Codes

- `0`: Success, no issues found
- `1`: Linting errors found
- `2`: Configuration error
- `3`: Runtime error
- `4`: Warnings found (with `--fail-on-warning`)

## Development

### Build

```bash
go build -o k8s-manifests-lint ./cmd/k8s-manifests-lint
```

### Run from source

```bash
go run ./cmd/k8s-manifests-lint run
```

### Test

```bash
go test ./...
```

## License

Apache License 2.0
