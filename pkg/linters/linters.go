package linters

import (
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/clusterrolebindingsecurity"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/healthprobes"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/imagetags"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/requiredlabels"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/resourcelimits"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/securitycontext"
)
