# Custom Linters

k8s-manifests-lint supports defining custom linters based on existing linter types. This allows you to create organization-specific rules without writing Go code.

## Overview

Custom linters are defined in your `.k8s-manifests-lint.yaml` configuration file under the `linters.custom` section. Each custom linter specifies:

- **name**: Unique identifier for the linter
- **description**: Human-readable description
- **type**: Base linter type to use (currently only `jq` is supported)
- **settings**: Configuration specific to the linter type

## JQ Linter

The `jq` linter type allows you to write custom validation rules using [jq](https://stedolan.github.io/jq/) expressions.

### Available Variables

JQ expressions have access to two variables:

- `$objects` - Array of all Kubernetes objects being linted
- `$object` - The current object being evaluated

### Configuration

```yaml
linters:
  custom:
    - name: my-custom-linter
      description: Description of what this linter checks
      type: jq
      settings:
        rules:
          - expression: <jq expression>
            message: <error message>
            severity: error|warning|info|fatal
            field: <optional field path>
            suggestion: <optional suggestion>
```

### Rule Fields

- **expression** (required): JQ expression that returns `true` if the issue should be reported
- **message** (required): Error message to display
- **severity** (optional): Issue severity level (default: `error`)
  - `fatal` - Critical issues that must be fixed
  - `error` - Errors that should be fixed
  - `warning` - Warnings that should be addressed
  - `info` - Informational messages
- **field** (optional): JSONPath to the problematic field
- **suggestion** (optional): Suggestion for fixing the issue

## Examples

### Check for Required Annotation

Ensure all resources have an `owner` annotation:

```yaml
linters:
  enable:
    - require-owner-annotation

  custom:
    - name: require-owner-annotation
      description: Ensures all resources have an owner annotation
      type: jq
      settings:
        rules:
          - expression: |
              $object | has("metadata") and
              (.metadata | has("annotations") and
              (.annotations | has("owner"))) | not
            message: Resource must have an 'owner' annotation
            severity: warning
            field: metadata.annotations.owner
            suggestion: Add metadata.annotations.owner to specify the resource owner
```

### Check for Node Affinity

Ensure deployments have node affinity configured:

```yaml
linters:
  enable:
    - require-node-affinity

  custom:
    - name: require-node-affinity
      description: Ensures deployments have node affinity configured
      type: jq
      settings:
        rules:
          - expression: |
              $object.kind == "Deployment" and
              ($object | .spec.template.spec.affinity.nodeAffinity // null | . == null)
            message: Deployment should have node affinity configured
            severity: info
            field: spec.template.spec.affinity.nodeAffinity
            suggestion: Consider adding node affinity to control pod placement
```

### Check for Specific Labels on Services

Ensure services have environment and tier labels:

```yaml
linters:
  enable:
    - service-labels

  custom:
    - name: service-labels
      description: Validates service labels
      type: jq
      settings:
        rules:
          - expression: |
              $object.kind == "Service" and
              ($object.metadata.labels.environment // "" | . == "")
            message: Service must have 'environment' label
            severity: error
            field: metadata.labels.environment
            suggestion: "Add label: environment: <prod|staging|dev>"

          - expression: |
              $object.kind == "Service" and
              ($object.metadata.labels.tier // "" | . == "")
            message: Service must have 'tier' label
            severity: error
            field: metadata.labels.tier
            suggestion: "Add label: tier: <frontend|backend|database>"
```

### Cross-Resource Validation

Check if a deployment references a ConfigMap that exists:

```yaml
linters:
  enable:
    - configmap-exists

  custom:
    - name: configmap-exists
      description: Ensures referenced ConfigMaps exist
      type: jq
      settings:
        rules:
          - expression: |
              $object.kind == "Deployment" and
              ($object.spec.template.spec.volumes[]? | select(.configMap) | .configMap.name) as $cmName |
              ($objects | map(select(.kind == "ConfigMap" and .metadata.name == $cmName)) | length == 0)
            message: Referenced ConfigMap does not exist
            severity: error
            field: spec.template.spec.volumes
            suggestion: Ensure the ConfigMap is defined or create it
```

### Validate Image Pull Policy

Ensure all containers use `IfNotPresent` or `Always` pull policy:

```yaml
linters:
  enable:
    - image-pull-policy

  custom:
    - name: image-pull-policy
      description: Validates image pull policy
      type: jq
      settings:
        rules:
          - expression: |
              ($object | .spec.template.spec.containers[]? // .spec.containers[]? |
              .imagePullPolicy // "" |
              select(. != "" and . != "IfNotPresent" and . != "Always")) | . != null
            message: Container must use 'IfNotPresent' or 'Always' image pull policy
            severity: warning
            field: spec.containers[].imagePullPolicy
            suggestion: Set imagePullPolicy to 'IfNotPresent' or 'Always'
```

## Usage

1. Define your custom linters in `.k8s-manifests-lint.yaml`
2. Enable them in the `linters.enable` list
3. Run the linter:

```bash
k8s-manifests-lint run
```

## Tips

### Testing JQ Expressions

You can test your jq expressions using the `jq` command-line tool or [jqplay.org](https://jqplay.org/):

```bash
# Test against a sample manifest
kubectl get deployment my-deployment -o json | jq '$object.kind == "Deployment"'
```

### Common Patterns

**Check if field exists:**
```jq
$object | .metadata.annotations.myfield // null | . == null
```

**Check if field has specific value:**
```jq
$object.metadata.labels.environment != "production"
```

**Check array elements:**
```jq
$object.spec.containers[] | select(.name == "nginx") | .image | test("latest$")
```

**Cross-resource check:**
```jq
$objects | map(select(.kind == "Service" and .metadata.name == "my-service")) | length == 0
```

## Limitations

- JQ expressions are evaluated for each object individually
- Complex cross-resource validations may impact performance
- Error messages from jq expression failures will reference the expression, not the YAML line

## Future Enhancements

Planned support for additional linter types:

- **cel**: Use Common Expression Language (CEL) for validation
- **rego**: Use Open Policy Agent (OPA) Rego for policy enforcement
- **python**: Use Python expressions for complex validation logic
