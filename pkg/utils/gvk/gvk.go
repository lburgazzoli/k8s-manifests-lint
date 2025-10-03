package gvk

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	Deployment = schema.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "Deployment",
	}

	StatefulSet = schema.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "StatefulSet",
	}

	DaemonSet = schema.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "DaemonSet",
	}

	Job = schema.GroupVersionKind{
		Group:   batchv1.SchemeGroupVersion.Group,
		Version: batchv1.SchemeGroupVersion.Version,
		Kind:    "Job",
	}

	CronJob = schema.GroupVersionKind{
		Group:   batchv1.SchemeGroupVersion.Group,
		Version: batchv1.SchemeGroupVersion.Version,
		Kind:    "CronJob",
	}

	Pod = schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Pod",
	}

	ClusterRoleBinding = schema.GroupVersionKind{
		Group:   rbacv1.SchemeGroupVersion.Group,
		Version: rbacv1.SchemeGroupVersion.Version,
		Kind:    "ClusterRoleBinding",
	}

	ConfigMap = schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ConfigMap",
	}

	Secret = schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Secret",
	}

	PodDisruptionBudget = schema.GroupVersionKind{
		Group:   policyv1.SchemeGroupVersion.Group,
		Version: policyv1.SchemeGroupVersion.Version,
		Kind:    "PodDisruptionBudget",
	}

	Service = schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Service",
	}

	ServiceMonitor = schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	}

	NetworkPolicy = schema.GroupVersionKind{
		Group:   networkingv1.SchemeGroupVersion.Group,
		Version: networkingv1.SchemeGroupVersion.Version,
		Kind:    "NetworkPolicy",
	}

	ResourceQuota = schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ResourceQuota",
	}
)

// IsGVK checks if an unstructured object matches the given GroupVersionKind
func IsGVK(obj unstructured.Unstructured, gvk schema.GroupVersionKind) bool {
	return obj.GroupVersionKind() == gvk
}

// IsAnyGVK checks if an unstructured object matches any of the given GroupVersionKinds
func IsAnyGVK(obj unstructured.Unstructured, gvks ...schema.GroupVersionKind) bool {
	objGVK := obj.GroupVersionKind()
	for _, gvk := range gvks {
		if objGVK == gvk {
			return true
		}
	}
	return false
}

// IsWorkload checks if an object is a workload resource (Deployment, StatefulSet, DaemonSet, Job, CronJob)
func IsWorkload(obj unstructured.Unstructured) bool {
	return IsAnyGVK(obj, Deployment, StatefulSet, DaemonSet, Job, CronJob)
}

// IsWorkloadOrPod checks if an object is a workload resource or Pod
func IsWorkloadOrPod(obj unstructured.Unstructured) bool {
	return IsAnyGVK(obj, Deployment, StatefulSet, DaemonSet, Job, CronJob, Pod)
}
