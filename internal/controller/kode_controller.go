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
	"reflect"
	"time"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Configuration constants
const (
	ContainerRestartPolicyAlways corev1.ContainerRestartPolicy = "Always"
	PersistentVolumeClaimName                                  = "kode-pvc"
	LoopRetryTime                                              = 1 * time.Second
	OperatorName                                               = "kode-operator"
	InternalServicePort                                        = 3000
	ExternalServicePort                                        = 8000
)

type KodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=kodeclustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=kode.kode.jacero.io,resources=envoyproxyclusterconfig,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// TODO: When requeued (EnvoyProxyConfig, EnvoyProxyClusterConfig) or (KodeTemplate, KodeClusterTemplate), force reconcile of Kode resource
// TODO: Fix when storage config is added after the Kode instance has deployed the volume-mount is not added. It should recreate the deployment with the PVC
// TODO: Investigate and fix why is it triggering the reconcile loop four times?
// TODO: Improve the logging output of the Kode instance and sidecar containers so that it seamlessly integrates with the opentelemetry logging, using labels.

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("Reconcile")

	labels := map[string]string{}

	// Fetch the Kode resource
	log.Info("Fetching Kode resource", "Namespace", req.Namespace, "Name", req.Name)
	kode := &kodev1alpha1.Kode{}
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to fetch Kode resource", "Namespace", req.Namespace, "Name", req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Kode resource found", "Namespace", kode.Namespace, "Name", kode.Name)
	logKodeManifest(log, kode)

	// Fetch the secret if ExistingSecret is set
	var username, password string
	if kode.Spec.ExistingSecret != "" {
		secret := &corev1.Secret{}
		secretNamespace := kode.Namespace
		if err := r.Get(ctx, types.NamespacedName{Name: kode.Spec.ExistingSecret, Namespace: secretNamespace}, secret); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Secret not found, requeuing", "Namespace", secretNamespace, "Name", kode.Spec.ExistingSecret)
				return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
			}
			log.Error(err, "Failed to fetch Secret", "Namespace", secretNamespace, "Name", kode.Spec.ExistingSecret)
			return ctrl.Result{}, err
		}
		username = string(secret.Data["username"])
		password = string(secret.Data["password"])
	} else {
		username = kode.Spec.User
		password = kode.Spec.Password
	}

	// Validate references
	if err := r.validateReferences(kode); err != nil {
		log.Error(err, "Invalid references in Kode", "Namespace", kode.Namespace, "Name", kode.Name)
		return ctrl.Result{}, err
	}

	// Fetch the KodeTemplate or KodeClusterTemplate resource
	var sharedKodeTemplateSpec kodev1alpha1.SharedKodeTemplateSpec
	var sharedEnvoyProxyConfigSpec kodev1alpha1.SharedEnvoyProxyConfigSpec
	var templateVersion string
	var envoyProxyConfigVersion string
	labels["app.kubernetes.io/name"] = "kode-" + kode.Name
	labels["app.kubernetes.io/managed-by"] = OperatorName
	labels["kode.jacero.io/name"] = kode.Name

	if kode.Spec.TemplateRef.Kind == "KodeTemplate" {
		// If no namespace is provided, use the namespace of the Kode resource
		kodeTemplateNamespace := kode.Spec.TemplateRef.Namespace
		if kodeTemplateNamespace == "" {
			kodeTemplateNamespace = kode.Namespace
		}

		kodeTemplate := &kodev1alpha1.KodeTemplate{}
		if err := r.Get(ctx, types.NamespacedName{Name: kode.Spec.TemplateRef.Name, Namespace: kodeTemplateNamespace}, kodeTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("KodeTemplate resource not found, requeuing", "Namespace", kodeTemplateNamespace, "Name", kode.Spec.TemplateRef.Name)
				return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
			}
			log.Error(err, "Failed to fetch KodeTemplate", "Namespace", kodeTemplateNamespace, "Name", kode.Spec.TemplateRef.Name)
			return ctrl.Result{}, err
		}
		sharedKodeTemplateSpec = kodeTemplate.Spec.SharedKodeTemplateSpec
		templateVersion = kodeTemplate.ResourceVersion
		labels["kode-template.kode.jacero.io/name"] = kodeTemplate.Name
		log.Info("KodeTemplate resource found", "Namespace", kodeTemplateNamespace, "Name", kodeTemplate.Name)

		// Fetch the EnvoyProxyConfig if it exists
		if kodeTemplate.Spec.EnvoyProxyRef.Name != "" {
			// If no namespace is provided, use the namespace of the Kode resource
			envoyProxyConfigNamespace := kodeTemplate.Spec.EnvoyProxyRef.Namespace
			if envoyProxyConfigNamespace == "" {
				envoyProxyConfigNamespace = kode.Namespace
			}
			envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{}
			if err := r.Get(ctx, types.NamespacedName{Name: kodeTemplate.Spec.EnvoyProxyRef.Name, Namespace: envoyProxyConfigNamespace}, envoyProxyConfig); err != nil {
				if errors.IsNotFound(err) {
					log.Info("EnvoyProxyConfig resource not found, requeuing", "Namespace", envoyProxyConfigNamespace, "Name", kodeTemplate.Spec.EnvoyProxyRef.Name)
					return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
				}
				log.Error(err, "Failed to fetch EnvoyProxyConfig",
					"EnvoyProxy_Namespace", envoyProxyConfigNamespace,
					"EnvoyProxy_Name", kodeTemplate.Spec.EnvoyProxyRef.Name,
					"KodeTemplate_Namespace", kodeTemplate.Namespace,
					"KodeTemplate_Name", kodeTemplate.Name)
				return ctrl.Result{}, err
			}
			sharedEnvoyProxyConfigSpec = envoyProxyConfig.Spec.SharedEnvoyProxyConfigSpec
			envoyProxyConfigVersion = envoyProxyConfig.ResourceVersion
			labels["envoyproxy-config.kode.jacero.io/name"] = envoyProxyConfig.Name
			log.Info("EnvoyProxyConfig resource found",
				"EnvoyProxy_Namespace", envoyProxyConfigNamespace,
				"EnvoyProxy_Name", envoyProxyConfig.Name,
				"KodeTemplate_Namespace", kodeTemplate.Namespace,
				"KodeTemplate_Name", kodeTemplate.Name)
		}
	} else if kode.Spec.TemplateRef.Kind == "KodeClusterTemplate" {
		kodeClusterTemplate := &kodev1alpha1.KodeClusterTemplate{}
		if err := r.Get(ctx, types.NamespacedName{Name: kode.Spec.TemplateRef.Name, Namespace: kode.Spec.TemplateRef.Namespace}, kodeClusterTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("KodeClusterTemplate resource not found, requeuing", "Name", kode.Spec.TemplateRef.Name)
				return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
			}
			log.Error(err, "Failed to fetch KodeClusterTemplate", "Name", kode.Spec.TemplateRef.Name)
			return ctrl.Result{}, err
		}
		sharedKodeTemplateSpec = kodeClusterTemplate.Spec.SharedKodeTemplateSpec
		templateVersion = kodeClusterTemplate.ResourceVersion
		labels["kode-cluster-template.kode.jacero.io/name"] = kodeClusterTemplate.Name
		log.Info("KodeClusterTemplate resource found", "Name", kodeClusterTemplate.Name)

		// Fetch the EnvoyProxyClusterConfig if it exists
		if kodeClusterTemplate.Spec.EnvoyProxyRef.Name != "" {
			envoyProxyClusterConfig := &kodev1alpha1.EnvoyProxyClusterConfig{}
			if err := r.Get(ctx, types.NamespacedName{Name: kodeClusterTemplate.Spec.EnvoyProxyRef.Name, Namespace: kodeClusterTemplate.Spec.EnvoyProxyRef.Namespace}, envoyProxyClusterConfig); err != nil {
				if errors.IsNotFound(err) {
					log.Info("EnvoyProxyConfig resource not found, requeuing", "Name", kodeClusterTemplate.Spec.EnvoyProxyRef.Name)
					return ctrl.Result{RequeueAfter: LoopRetryTime}, nil
				}
				log.Error(err, "Failed to fetch EnvoyProxyClusterConfig",
					"EnvoyProxy_Name", kodeClusterTemplate.Spec.EnvoyProxyRef.Name,
					"KodeTemplate_Namespace", kodeClusterTemplate.Namespace,
					"KodeTemplate_Name", kodeClusterTemplate.Name)
				return ctrl.Result{}, err
			}
			sharedEnvoyProxyConfigSpec = envoyProxyClusterConfig.Spec.SharedEnvoyProxyConfigSpec
			envoyProxyConfigVersion = envoyProxyClusterConfig.ResourceVersion
			labels["envoyproxy-cluster-config.kode.jacero.io/name"] = envoyProxyClusterConfig.Name
			log.Info("EnvoyProxyClusterConfig resource found",
				"EnvoyProxy_Name", envoyProxyClusterConfig.Name,
				"KodeTemplate_Name", kodeClusterTemplate.Name)
		}
	}

	// Ensure the Deployment and Service exist
	if err := r.ensureDeployment(ctx, kode, labels, &sharedKodeTemplateSpec, &sharedEnvoyProxyConfigSpec, templateVersion, envoyProxyConfigVersion, username, password); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureService(ctx, kode, labels, &sharedKodeTemplateSpec); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure PVC exists
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		pvcResult, _, err := r.ensurePVC(ctx, kode)
		if err != nil {
			return ctrl.Result{}, err
		}

		switch pvcResult {
		case controllerutil.OperationResultCreated:
			r.Log.Info("PVC created", "Namespace", kode.Namespace, "Name", kode.Name)
		case controllerutil.OperationResultUpdated:
			r.Log.Info("PVC updated", "Namespace", kode.Namespace, "Name", kode.Name)
		case controllerutil.OperationResultNone:
			r.Log.Info("PVC unchanged", "Namespace", kode.Namespace, "Name", kode.Name)
		}
	}

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) validateReferences(kode *kodev1alpha1.Kode) error {
	if kode.Spec.TemplateRef.Kind != "KodeTemplate" && kode.Spec.TemplateRef.Kind != "KodeClusterTemplate" {
		return fmt.Errorf("invalid reference kind for TemplateRef: %s", kode.Spec.TemplateRef.Kind)
	}
	return nil
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("Kode")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		// Watches(&kodev1alpha1.KodeTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.KodeClusterTemplate{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.EnvoyProxyConfig{}, &handler.EnqueueRequestForObject{}).
		// Watches(&kodev1alpha1.EnvoyProxyClusterConfig{}, &handler.EnqueueRequestForObject{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
