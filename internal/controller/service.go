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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureService ensures that the Service exists for the Kode instance
func (r *KodeReconciler) ensureService(ctx context.Context, kode *kodev1alpha1.Kode,
	labels map[string]string,
	sharedKodeTemplateSpec *kodev1alpha1.SharedKodeTemplateSpec) error {

	log := r.Log.WithName("ensureService")
	log.Info("Ensuring Service exists", "Namespace", kode.Namespace, "Name", kode.Name)

	service := r.constructService(kode, labels, sharedKodeTemplateSpec)
	if err := controllerutil.SetControllerReference(kode, service, r.Scheme); err != nil {
		return err
	}

	// Use controllerutil.CreateOrUpdate for idempotency
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		// Update service spec to ensure correct state
		service.Spec.Selector = labels
		service.Spec.Ports = []corev1.ServicePort{{
			Protocol:   corev1.ProtocolTCP,
			Port:       sharedKodeTemplateSpec.Port,
			TargetPort: intstr.FromInt(int(sharedKodeTemplateSpec.Port)),
		}}
		return nil
	})
	if err != nil {
		log.Error(err, "Failed to create or update Service", "Namespace", service.Namespace, "Name", service.Name)
		return err
	}
	log.Info("Service ensured", "operation", op, "Namespace", service.Namespace, "Name", service.Name)
	return nil
}

// constructService constructs a Service for the Kode instance
func (r *KodeReconciler) constructService(kode *kodev1alpha1.Kode,
	labels map[string]string,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec) *corev1.Service {

	log := r.Log.WithName("constructService")
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       templateSpec.Port,
				TargetPort: intstr.FromInt(int(templateSpec.Port)),
			}},
		},
	}
	logServiceManifest(log, service)
	return service
}
