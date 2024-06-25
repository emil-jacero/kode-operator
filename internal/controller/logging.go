package controller

import (
	"fmt"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// mask function to mask sensitive values
func mask(s string) string {
	return "******"
}

// maskEnvs function to mask sensitive environment variables
func maskEnvs(envs []corev1.EnvVar) []corev1.EnvVar {
	maskedEnvs := make([]corev1.EnvVar, len(envs))
	for i, env := range envs {
		if env.Name == "PASSWORD" || env.Name == "SECRET" {
			env.Value = mask(env.Value)
		}
		maskedEnvs[i] = env
	}
	return maskedEnvs
}

// logKodeManifest add structured logging for the Kode manifest to improve visibility for debugging
func logKodeManifest(log logr.Logger, kode *kodev1alpha1.Kode) {
	log.V(1).Info("Kode Manifest",
		"Name", kode.Name,
		"Namespace", kode.Namespace,
		"TemplateRef", kode.Spec.TemplateRef,
		"User", kode.Spec.User,
		"Password", mask(kode.Spec.Password),
		"ExistingSecret", kode.Spec.ExistingSecret,
		"Storage", fmt.Sprintf("%v", kode.Spec.Storage),
		"Home", kode.Spec.Home,
		"Workspace", kode.Spec.Workspace,
		"UserConfig", kode.Spec.UserConfig,
		"Privileged", *kode.Spec.Privileged,
		"InitPlugins", fmt.Sprintf("%v", kode.Spec.InitPlugins),
	)
}

// logSharedKodeTemplateManifest add structured logging for the Kode Template manifest to improve visibility for debugging
func logSharedKodeTemplateManifest(log logr.Logger, name string, namespace string, spec kodev1alpha1.SharedKodeTemplateSpec) {
	maskedEnvs := maskEnvs(spec.Envs)
	log.V(1).Info("Kode Template Manifest",
		"Name", name,
		"Namespace", namespace,
		"Image", spec.Image,
		"TZ", spec.TZ,
		"PUID", spec.PUID,
		"PGID", spec.PGID,
		"Port", spec.Port,
		"Envs", fmt.Sprintf("%v", maskedEnvs),
		"Args", fmt.Sprintf("%v", spec.Args),
		"Home", spec.DefaultHome,
		"DefaultWorkspace", spec.DefaultWorkspace,
		"AllowPrivileged", spec.AllowPrivileged,
		"InitPlugins", fmt.Sprintf("%v", spec.InitPlugins),
		"InactiveAfterSeconds", spec.InactiveAfterSeconds,
		"RecycleAfterSeconds", spec.RecycleAfterSeconds,
	)
}

// logSharedEnvoyProxyTemplateManifest add structured logging for the Envoy Proxy Template manifest to improve visibility for debugging
func logSharedEnvoyProxyTemplateManifest(log logr.Logger, name string, namespace string, spec *kodev1alpha1.EnvoyProxyConfigSpec) {
	log.V(1).Info("Envoy Proxy Template Manifest",
		"Name", name,
		"Namespace", namespace,
		"Image", spec.Image,
		"AuthType", spec.AuthType,
		"HTTPFilters", fmt.Sprintf("%v", spec.HTTPFilters),
		"Clusters", fmt.Sprintf("%v", spec.Clusters),
	)
}

// logDeploymentManifest add structured logging for the Deployment manifest to improve visibility for debugging
func logDeploymentManifest(log logr.Logger, deployment *appsv1.Deployment) {
	maskedEnvs := maskEnvs(deployment.Spec.Template.Spec.Containers[0].Env)
	log.V(1).Info("Deployment Manifest",
		"Name", deployment.Name,
		"Namespace", deployment.Namespace,
		"Replicas", *deployment.Spec.Replicas,
		"Image", deployment.Spec.Template.Spec.Containers[0].Image,
		"Ports", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].Ports),
		"Env", fmt.Sprintf("%v", maskedEnvs),
		"VolumeMounts", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].VolumeMounts),
		"Volumes", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Volumes),
	)
}

// logPVCManifest add structured logging for the PVC manifest to improve visibility for debugging
func logPVCManifest(log logr.Logger, pvc *corev1.PersistentVolumeClaim) {
	log.V(1).Info("PVC Manifest",
		"Name", pvc.Name,
		"Namespace", pvc.Namespace,
		"AccessModes", fmt.Sprintf("%v", pvc.Spec.AccessModes),
		"Resources", fmt.Sprintf("Requests: %v, Limits: %v", pvc.Spec.Resources.Requests, pvc.Spec.Resources.Limits),
		"StorageClassName", pvc.Spec.StorageClassName,
	)
}

// logServiceManifest add structured logging for the Service manifest to improve visibility for debugging
func logServiceManifest(log logr.Logger, service *corev1.Service) {
	log.V(1).Info("Service Manifest",
		"Name", service.Name,
		"Namespace", service.Namespace,
		"Selector", fmt.Sprintf("%v", service.Spec.Selector),
		"Ports", fmt.Sprintf("%v", service.Spec.Ports),
	)
}
