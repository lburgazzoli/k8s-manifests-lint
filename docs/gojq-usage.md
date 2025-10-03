# Using gojq in Linters

The `pkg/k8s/query.go` package provides gojq-based helpers for querying unstructured Kubernetes objects. This makes linter implementations cleaner and more maintainable.

## Available Query Functions

### Basic Queries

```go
// Query executes a jq-style query and returns the result
value, err := k8s.Query(obj, ".spec.replicas")

// QueryString returns a string value
name, found, err := k8s.QueryString(obj, ".metadata.name")

// QueryBool returns a boolean value
enabled, found, err := k8s.QueryBool(obj, ".spec.enabled")

// QueryInt returns an integer value
replicas, found, err := k8s.QueryInt(obj, ".spec.replicas")

// QueryExists checks if a field exists
exists, err := k8s.QueryExists(obj, ".spec.template.spec.containers")
```

### Array Queries

```go
// QueryArray returns all matching results as a slice
results, err := k8s.QueryArray(obj, `.subjects[] | select(.kind == "Group") | .name`)
```

### Helper Functions

```go
// GetContainers handles different resource types (Pod, Deployment, CronJob, etc.)
containers, err := k8s.GetContainers(obj)
```

## Example: Before and After

### Before (using unstructured helpers)

```go
func (l *ClusterRoleBindingSecurityLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
    // Get roleRef
    roleRef, found, err := unstructured.NestedMap(obj.Object, "roleRef")
    if err != nil || !found {
        return nil, nil
    }
    roleName, _ := roleRef["name"].(string)

    // Get subjects
    subjects, found, err := unstructured.NestedSlice(obj.Object, "subjects")
    if err != nil || !found {
        return nil, nil
    }

    // Filter for Group subjects
    for _, subject := range subjects {
        subjectMap, ok := subject.(map[string]interface{})
        if !ok {
            continue
        }

        kind, _ := subjectMap["kind"].(string)
        name, _ := subjectMap["name"].(string)

        if kind != "Group" {
            continue
        }

        // Check the group name...
    }
}
```

### After (using gojq)

```go
func (l *ClusterRoleBindingSecurityLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
    // Get role name using gojq
    roleName, _, err := k8s.QueryString(obj, ".roleRef.name")
    if err != nil {
        return nil, err
    }

    // Get all Group subjects using gojq - filter in one query!
    groupSubjects, err := k8s.QueryArray(obj, `.subjects[] | select(.kind == "Group") | .name`)
    if err != nil {
        return nil, err
    }

    for _, subject := range groupSubjects {
        name, ok := subject.(string)
        if !ok {
            continue
        }

        // Check the group name...
    }
}
```

## Benefits

1. **More Readable**: jq syntax is concise and expressive
2. **Less Boilerplate**: No need for multiple nested type assertions
3. **Powerful Filtering**: Filter arrays inline with `select()`
4. **Consistent**: Same query language across all linters
5. **Testable**: Easy to test queries in isolation

## Common Query Patterns

### Get a nested field

```go
// Simple path
value, _ := k8s.QueryString(obj, ".metadata.name")

// Nested path
value, _ := k8s.QueryString(obj, ".spec.template.metadata.labels.app")
```

### Get array elements

```go
// All container names
names, _ := k8s.QueryArray(obj, ".spec.template.spec.containers[].name")

// Filtered containers
images, _ := k8s.QueryArray(obj, `.spec.template.spec.containers[] | select(.name == "nginx") | .image`)
```

### Check for existence

```go
// Check if field exists
hasProbe, _ := k8s.QueryExists(obj, ".spec.template.spec.containers[0].livenessProbe")
```

### Complex filtering

```go
// Get names of all Group subjects that start with "system:"
groups, _ := k8s.QueryArray(obj, `.subjects[] | select(.kind == "Group" and (.name | startswith("system:"))) | .name`)
```

## Writing New Linters with gojq

When writing a new linter:

1. Use `k8s.QueryString()`, `k8s.QueryBool()`, etc. for simple field access
2. Use `k8s.QueryArray()` with filters for complex array operations
3. Use `k8s.GetContainers()` for workload resources
4. Test your jq queries independently before integrating

Example template:

```go
func (l *MyLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
    // Skip irrelevant kinds
    if obj.GetKind() != "MyKind" {
        return nil, nil
    }

    var issues []linter.Issue

    // Use gojq to query fields
    value, found, err := k8s.QueryString(obj, ".spec.myField")
    if err != nil {
        return nil, err
    }

    if !found || value != "expected" {
        issues = append(issues, linter.Issue{
            Severity:   linter.SeverityError,
            Linter:     l.Name(),
            Message:    "Field has incorrect value",
            Resource:   resourceRef(obj),
            Field:      "spec.myField",
            Suggestion: "Set to: expected",
        })
    }

    return issues, nil
}
```
