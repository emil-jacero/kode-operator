/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"path"
	"reflect"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureDeployment ensures that the Deployment exists for the Kode instance
func (r *KodeReconciler) ensureDeployment(ctx context.Context,
	kode *kodev1alpha1.Kode,
	labels map[string]string,
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyConfigSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec) error {

	log := r.Log.WithName("ensureDeployment")

	log.Info("Ensuring Deployment exists", "Namespace", kode.Namespace, "Name", kode.Name)

	deployment := r.constructDeployment(kode, labels, sharedKodeTemplateSpec, sharedEnvoyProxyConfigSpec)
	if err := controllerutil.SetControllerReference(kode, deployment, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			if err := r.Create(ctx, deployment); err != nil {
				log.Error(err, "Failed to create Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
				return err
			}
			log.Info("Deployment created", "Namespace", deployment.Namespace, "Name", deployment.Name)
		} else {
			log.Error(err, "Failed to get Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			return err
		}
	} else if !reflect.DeepEqual(deployment.Spec, found.Spec) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found); err != nil {
				return err
			}
			found.Spec = deployment.Spec
			log.Info("Updating Deployment due to spec change", "Namespace", found.Namespace, "Name", found.Name)
			return r.Update(ctx, found)
		})

		if retryErr != nil {
			log.Error(retryErr, "Failed to update Deployment after retrying", "Namespace", deployment.Namespace, "Name", deployment.Name)
			return retryErr
		}
	}

	log.Info("Successfully ensured Deployment", "Namespace", kode.Namespace, "Name", kode.Name)
	return nil
}

// constructDeployment constructs a Deployment for the Kode instance
func (r *KodeReconciler) constructDeployment(kode *kodev1alpha1.Kode,
	labels map[string]string,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	sharedEnvoyProxyConfigSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec) *appsv1.Deployment {

	log := r.Log.WithName("constructDeployment")

	replicas := int32(1)

	var workspace string
	var mountPath string

	workspace = path.Join(templateSpec.DefaultHome, templateSpec.DefaultWorkspace)
	mountPath = templateSpec.DefaultHome
	if kode.Spec.Workspace != "" {
		if kode.Spec.Home != "" {
			workspace = path.Join(kode.Spec.Home, kode.Spec.Workspace)
			mountPath = kode.Spec.Home
		} else {
			workspace = path.Join(templateSpec.DefaultHome, kode.Spec.Workspace)
		}
	}

	var containers []corev1.Container
	var initContainers []corev1.Container

	if templateSpec.Type == "code-server" {
		containers = constructCodeServerContainers(kode, templateSpec, workspace)
	} else if templateSpec.Type == "webtop" {
		containers = constructWebtopContainers(kode, templateSpec)
	}

	volumes, volumeMounts := constructVolumesAndMounts(mountPath, kode)
	containers[0].VolumeMounts = volumeMounts

	if templateSpec.EnvoyProxyRef.Name != "" {
		log.Info("EnvoyProxyRef is defined", "Namespace", kode.Namespace, "Kode", kode.Name, "Name", templateSpec.EnvoyProxyRef.Name)
		envoySidecarContainer, envoyInitContainer, err := constructEnvoyProxyContainer(log, templateSpec, sharedEnvoyProxyConfigSpec)
		if err != nil {
			log.Error(err, "Failed to construct EnvoyProxy sidecar", "Kode", kode.Name, "Container", templateSpec.EnvoyProxyRef.Name, "Error", err)
		} else {
			containers = append(containers, envoySidecarContainer)
			initContainers = append(initContainers, envoyInitContainer)
			log.Info("Added EnvoyProxy sidecar container and init container", "Kode", kode.Name, "Container", envoySidecarContainer.Name)
		}
	}

	// Add InitPlugins as InitContainers
	for _, initPlugin := range kode.Spec.InitPlugins {
		initContainers = append(initContainers, constructInitPluginContainer(initPlugin))
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers:     containers,
					Volumes:        volumes,
				},
			},
		},
	}

	logDeploymentManifest(log, deployment)
	return deployment
}

func constructCodeServerContainers(kode *kodev1alpha1.Kode,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	workspace string) []corev1.Container {

	return []corev1.Container{{
		Name:  "kode-" + kode.Name,
		Image: templateSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PORT", Value: fmt.Sprintf("%d", templateSpec.Port)},
			{Name: "PUID", Value: fmt.Sprintf("%d", templateSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", templateSpec.PGID)},
			{Name: "TZ", Value: templateSpec.TZ},
			{Name: "USERNAME", Value: kode.Spec.User},
			{Name: "PASSWORD", Value: kode.Spec.Password},
			{Name: "DEFAULT_WORKSPACE", Value: workspace},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: templateSpec.Port,
		}},
	}}
}

func constructWebtopContainers(kode *kodev1alpha1.Kode,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec) []corev1.Container {

	return []corev1.Container{{
		Name:  "kode-" + kode.Name,
		Image: templateSpec.Image,
		Env: []corev1.EnvVar{
			{Name: "PUID", Value: fmt.Sprintf("%d", templateSpec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", templateSpec.PGID)},
			{Name: "TZ", Value: templateSpec.TZ},
			{Name: "CUSTOM_PORT", Value: fmt.Sprintf("%d", templateSpec.Port)},
			{Name: "CUSTOM_USER", Value: kode.Spec.User},
			{Name: "PASSWORD", Value: kode.Spec.Password},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: templateSpec.Port,
		}},
	}}
}

func constructVolumesAndMounts(mountPath string, kode *kodev1alpha1.Kode) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add volume and volume mount if storage is defined
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		volume := corev1.Volume{
			Name: "kode-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName,
				},
			},
		}

		volumeMount := corev1.VolumeMount{
			Name:      "kode-storage",
			MountPath: mountPath,
		}

		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumes, volumeMounts
}

func constructInitPluginContainer(plugin kodev1alpha1.InitPluginSpec) corev1.Container {
	return corev1.Container{
		Name:    "init-" + plugin.Image,
		Image:   plugin.Image,
		Command: plugin.Command,
		Args:    plugin.Args,
		Env:     plugin.EnvVars,
	}
}
