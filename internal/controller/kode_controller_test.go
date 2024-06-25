package controller

// STANDARD TESTS (using KodeTemplate and EnvoyProxyConfig)
// TEST: It should create a Deployment for the Kode resource
// TEST: It should create a Service for the Kode resource
// TEST: It should create a PersistentVolumeClaim for the Kode resource if storage is defined
// TEST: It should handle missing required fields gracefully
// TEST: It should handle an invalid image name

// ADVANCED TESTS (using KodeTemplate and EnvoyProxyConfig)
// TEST: It should create a Kode resource if the TemplateRef is set and the KodeTemplate exists
// TEST: It should reconcile the Kode resource when the KodeTemplate is updated
// TEST: It should reconcile the Kode resource when the EnvoyProxyConfig is updated
// TEST: It should reconcile the Kode resource when the KodeTemplate is deleted
// TEST: It should NOT reconcile the Kode resource when the EnvoyProxyConfig is deleted
// TEST: It should create a Kode resource if the KodeTemplate.Spec.EnvoyProxyRef is set and the EnvoyProxyConfig exists
// TEST: It should NOT create a Kode resource if the KodeTemplate.Spec.EnvoyProxyRef is set and the EnvoyProxyConfig does not exist
// TEST: It should create a Kode resource without an envoy proxy sidecar when the KodeTemplate.Spec.EnvoyProxyRef is not set
// TEST: It should create a Kode resource with an envoy proxy sidecar when the KodeTemplate.Spec.EnvoyProxyRef is set
// TEST: It should add InitPlugins to the Deployment when the KodeTemplate has InitPlugins and the Kode resource has InitPlugins. The order of the InitPlugins should be preserved. KodeTemplate comes first, then Kode resource.

// STORAGE TESTS (using KodeTemplate and EnvoyProxyConfig)
// TEST: It should create a PersistentVolumeClaim for the Kode resource with storage configured
// TEST: It should update the PersistentVolumeClaim when the storage configuration changes
// TEST: It should NOT create a PersistentVolumeClaim for the Kode resource without storage configured
// TEST: It should NOT delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to true
// TEST: It should delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to false

// SECURITY TESTS (using KodeTemplate and EnvoyProxyConfig)
// TEST: It should create a Kode resource with the specified user and password, using HTTP Basic auth
// TEST: It should create a Kode resource with the specified user and password from an existing secret, using HTTP Basic auth
// TEST: It should create a Kode resource with the specified home directory
// TEST: It should create a Kode resource with the specified workspace directory
// TEST: It should create a Kode resource with only the default KodeTemplate fields

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// STANDARD TESTS (using KodeClusterTemplate and EnvoyProxyClusterConfig)
// TEST: It should create a Deployment for the Kode resource
// TEST: It should create a Service for the Kode resource
// TEST: It should create a PersistentVolumeClaim for the Kode resource if storage is defined
// TEST: It should handle missing required fields gracefully
// TEST: It should handle an invalid image name

// ADVANCED TESTS (using KodeClusterTemplate and EnvoyProxyClusterConfig)
// TEST: It should create a Kode resource if the TemplateRef is set and the KodeClusterTemplate exists
// TEST: It should reconcile the Kode resource when the KodeClusterTemplate is updated
// TEST: It should reconcile the Kode resource when the EnvoyProxyClusterConfig is updated
// TEST: It should reconcile the Kode resource when the KodeClusterTemplate is deleted
// TEST: It should NOT reconcile the Kode resource when the EnvoyProxyClusterConfig is deleted
// TEST: It should create a Kode resource if the KodeClusterTemplate.Spec.EnvoyProxyRef is set and the EnvoyProxyClusterConfig exists
// TEST: It should NOT create a Kode resource if the KodeClusterTemplate.Spec.EnvoyProxyRef is set and the EnvoyProxyClusterConfig does not exist
// TEST: It should create a Kode resource without an envoy proxy sidecar when the KodeClusterTemplate.Spec.EnvoyProxyRef is not set
// TEST: It should create a Kode resource with an envoy proxy sidecar when the KodeClusterTemplate.Spec.EnvoyProxyRef is set

// ADVANCED TESTS (using KodeClusterTemplate and EnvoyProxyClusterConfig)
// TEST: It should create a PersistentVolumeClaim for the Kode resource with storage configured
// TEST: It should update the PersistentVolumeClaim when the storage configuration changes
// TEST: It should NOT create a PersistentVolumeClaim for the Kode resource without storage configured
// TEST: It should NOT delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to true
// TEST: It should delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to false

// STORAGE TESTS (using KodeTemplate and EnvoyProxyConfig)
// TEST: It should create a PersistentVolumeClaim for the Kode resource with storage configured
// TEST: It should update the PersistentVolumeClaim when the storage configuration changes
// TEST: It should NOT create a PersistentVolumeClaim for the Kode resource without storage configured
// TEST: It should NOT delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to true
// TEST: It should delete the PersistentVolumeClaim when the Kode resource is deleted if the option KeepVolume is set to false

// SECURITY TESTS (using KodeClusterTemplate and EnvoyProxyClusterConfig)
// TEST: It should create a Kode resource with the specified user and password, using HTTP Basic auth
// TEST: It should create a Kode resource with the specified user and password from an existing secret, using HTTP Basic auth
// TEST: It should create a Kode resource with the specified home directory
// TEST: It should create a Kode resource with the specified workspace directory
// TEST: It should create a Kode resource with only the default KodeTemplate fields

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 20
	interval = time.Millisecond * 250
)

var _ = Describe("Kode Controller", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		testEnv    *envtest.Environment
		k8sClient  client.Client
		k8sManager ctrl.Manager
		reconciler *KodeReconciler
	)

	BeforeEach(func() {
		By("bootstrapping test environment")
		ctx, cancel = context.WithCancel(context.Background())
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		err = kodev1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		k8sClient = k8sManager.GetClient()
		Expect(k8sClient).ToNot(BeNil())

		reconciler = &KodeReconciler{
			Client: k8sClient,
			Scheme: k8sManager.GetScheme(),
			Log:    ctrl.Log.WithName("controllers").WithName("Kode"),
		}

		err = reconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(ctx)).To(Succeed())
		}()
	})

	AfterEach(func() {
		cancel()
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When reconciling a Kode resource", func() {
		const (
			resourceNamespace      = "test-namespace"
			kodeResourceName       = "test-kode"
			kodeTemplateKind       = "KodeTemplate"
			kodeTemplateName       = "test-kodetemplate"
			kodeTemplateImage      = "lscr.io/linuxserver/code-server:latest"
			envoyProxyConfigKind   = "EnvoyProxyConfig"
			envoyProxyConfigName   = "test-envoyproxyconfig"
			envoyProxyConfigImage  = "envoyproxy/envoy:v1.30-latest"
			envoyProxyConfigFilter = `{
				"@type":"type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz",
				"with_request_body":{
					"max_request_bytes":8192,
					"allow_partial_message":true
				},
				"failure_mode_allow":false,
				"grpc_service":{
					"envoy_grpc":{
						"cluster_name":"ext_authz_server"
					},
					"timeout":"0.5s"
				},
				"transport_api_version":"v3"
			}`
		)

		typeNamespacedName := types.NamespacedName{
			Name:      kodeResourceName,
			Namespace: resourceNamespace,
		}

		BeforeEach(func() {
			By("creating the namespace for the test")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceNamespace,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeResourceName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind:      kodeTemplateKind,
						Name:      kodeTemplateName,
						Namespace: resourceNamespace,
					},
					Storage: kodev1alpha1.KodeStorageSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			}
			err = k8sClient.Create(ctx, kode)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind KodeTemplate")
			kodeTemplate := &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeTemplateName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
						EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
							Kind: envoyProxyConfigKind,
							Name: envoyProxyConfigName,
						},
						Image: kodeTemplateImage,
						Type:  "code-server",
						Port:  8000,
					},
				},
			}
			err = k8sClient.Create(ctx, kodeTemplate)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind EnvoyProxyConfig")
			envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      envoyProxyConfigName,
					Namespace: resourceNamespace,
				},
				Spec: kodev1alpha1.EnvoyProxyConfigSpec{
					SharedEnvoyProxyConfigSpec: kodev1alpha1.SharedEnvoyProxyConfigSpec{
						Image: envoyProxyConfigImage,
						HTTPFilters: []kodev1alpha1.HTTPFilter{{
							Name:        "filter1",
							TypedConfig: runtime.RawExtension{Raw: []byte(envoyProxyConfigFilter)},
						}},
					},
				},
			}
			err = k8sClient.Create(ctx, envoyProxyConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("deleting the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{}
			err := k8sClient.Get(ctx, typeNamespacedName, kode)
			if err == nil {
				Expect(k8sClient.Delete(ctx, kode)).To(Succeed())
			}

			By("deleting the custom resource for the Kind KodeTemplate")
			kodeTemplate := &kodev1alpha1.KodeTemplate{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: resourceNamespace, Name: kodeTemplateName}, kodeTemplate)
			if err == nil {
				Expect(k8sClient.Delete(ctx, kodeTemplate)).To(Succeed())
			}

			By("deleting the custom resource for the Kind EnvoyProxyConfig")
			envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: resourceNamespace, Name: envoyProxyConfigName}, envoyProxyConfig)
			if err == nil {
				Expect(k8sClient.Delete(ctx, envoyProxyConfig)).To(Succeed())
			}

			By("deleting the namespace for the test")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceNamespace,
				},
			}
			err = k8sClient.Delete(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a Deployment for the Kode resource", func() {
			By("checking if the Deployment has been created")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, deployment)
			}, timeout, interval).Should(Succeed())

			Expect(deployment.Name).To(Equal(kodeResourceName))
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(kodeTemplateImage))
		})

		It("should create a Service for the Kode resource", func() {
			By("checking if the Service has been created")
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, service)
			}, timeout, interval).Should(Succeed())

			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8000)))
		})

		// // TODO: Fix validation. It should find the created PVC
		// It("should create a PersistentVolumeClaim for the Kode resource if storage is defined", func() {
		// 	By("checking if the PersistentVolumeClaim has been created")
		// 	pvcNamespacedName := types.NamespacedName{
		// 		Name:      "kode-pvc",
		// 		Namespace: resourceNamespace,
		// 	}
		// 	pvc := &corev1.PersistentVolumeClaim{}
		// 	Eventually(func() error {
		// 		return k8sClient.Get(ctx, pvcNamespacedName, pvc)
		// 	}, timeout, interval).Should(Succeed())

		// 	Expect(pvc.Name).To(Equal("kode-pvc"))
		// 	Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
		// })

		// // TODO: Fix validation. It should fail when KodeTemplate has an invalid image name
		// It("should handle an invalid image name", func() {
		// 	By("creating a KodeTemplate resource with an invalid image name")
		// 	kodeTemplate := &kodev1alpha1.KodeTemplate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "test-kodetemplate-invalid",
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.KodeTemplateSpec{
		// 			SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
		// 				EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
		// 					Kind: envoyProxyConfigKind,
		// 					Name: envoyProxyConfigName,
		// 				},
		// 				Image: "invalid-image-name",
		// 				Type:  "code-server",
		// 			},
		// 		},
		// 	}
		// 	err := k8sClient.Create(ctx, kodeTemplate)
		// 	Expect(err).ToNot(HaveOccurred())

		// 	By("creating a Kode resource that references the invalid KodeTemplate")
		// 	kode := &kodev1alpha1.Kode{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "test-kode-invalid-image",
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.KodeSpec{
		// 			TemplateRef: kodev1alpha1.KodeTemplateReference{
		// 				Kind:      kodeTemplateKind,
		// 				Name:      "test-kodetemplate-invalid",
		// 				Namespace: resourceNamespace,
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, kode)
		// 	Expect(err).To(HaveOccurred())
		// })

		It("should create a Kode resource if the TemplateRef is set and the KodeTemplate exists", func() {
			testNamespace := "test-create-if-template-exists"

			By("creating the namespace for the test")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind KodeTemplate")
			kodeTemplate := &kodev1alpha1.KodeTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeTemplateName,
					Namespace: namespace.Name,
				},
				Spec: kodev1alpha1.KodeTemplateSpec{
					SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
						EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
							Kind: envoyProxyConfigKind,
							Name: envoyProxyConfigName,
						},
						Image: kodeTemplateImage,
						Type:  "code-server",
					},
				},
			}
			err = k8sClient.Create(ctx, kodeTemplate)
			Expect(err).NotTo(HaveOccurred())
			// Eventually(func() error {
			// 	return k8sClient.Get(ctx, typeNamespacedName, kodeTemplate)
			// }, timeout, interval).Should(Succeed())

			By("creating the custom resource for the Kind EnvoyProxyConfig")
			envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      envoyProxyConfigName,
					Namespace: namespace.Name,
				},
				Spec: kodev1alpha1.EnvoyProxyConfigSpec{
					SharedEnvoyProxyConfigSpec: kodev1alpha1.SharedEnvoyProxyConfigSpec{
						Image: envoyProxyConfigImage,
						HTTPFilters: []kodev1alpha1.HTTPFilter{{
							Name:        "filter1",
							TypedConfig: runtime.RawExtension{Raw: []byte(envoyProxyConfigFilter)},
						}},
					},
				},
			}
			err = k8sClient.Create(ctx, envoyProxyConfig)
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind Kode")
			kode := &kodev1alpha1.Kode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kodeResourceName,
					Namespace: namespace.Name,
				},
				Spec: kodev1alpha1.KodeSpec{
					TemplateRef: kodev1alpha1.KodeTemplateReference{
						Kind:      kodeTemplateKind,
						Name:      kodeTemplateName,
						Namespace: namespace.Name,
					},
					Storage: kodev1alpha1.KodeStorageSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			}
			err = k8sClient.Create(ctx, kode)
			Expect(err).NotTo(HaveOccurred())
		})

		// It("should reconcile the Kode resource when the KodeTemplate is updated", func() {
		// 	testNamespace := "test-reconcile-if-template-updated"

		// 	By("creating the namespace for the test")
		// 	namespace := &corev1.Namespace{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name: testNamespace,
		// 		},
		// 	}
		// 	err := k8sClient.Create(ctx, namespace)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("creating the custom resource for the Kind KodeTemplate")
		// 	kodeTemplate := &kodev1alpha1.KodeTemplate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      kodeTemplateName,
		// 			Namespace: namespace.Name,
		// 		},
		// 		Spec: kodev1alpha1.KodeTemplateSpec{
		// 			SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
		// 				EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
		// 					Kind: envoyProxyConfigKind,
		// 					Name: envoyProxyConfigName,
		// 				},
		// 				Image: kodeTemplateImage,
		// 				Type:  "code-server",
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, kodeTemplate)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("creating the custom resource for the Kind EnvoyProxyConfig")
		// 	envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      envoyProxyConfigName,
		// 			Namespace: namespace.Name,
		// 		},
		// 		Spec: kodev1alpha1.EnvoyProxyConfigSpec{
		// 			SharedEnvoyProxyConfigSpec: kodev1alpha1.SharedEnvoyProxyConfigSpec{
		// 				Image: envoyProxyConfigImage,
		// 				HTTPFilters: []kodev1alpha1.HTTPFilter{{
		// 					Name:        "filter1",
		// 					TypedConfig: runtime.RawExtension{Raw: []byte(envoyProxyConfigFilter)},
		// 				}},
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, envoyProxyConfig)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("creating the custom resource for the Kind Kode")
		// 	kode := &kodev1alpha1.Kode{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      kodeResourceName,
		// 			Namespace: namespace.Name,
		// 		},
		// 		Spec: kodev1alpha1.KodeSpec{
		// 			TemplateRef: kodev1alpha1.KodeTemplateReference{
		// 				Kind:      kodeTemplateKind,
		// 				Name:      kodeTemplateName,
		// 				Namespace: namespace.Name,
		// 			},
		// 			Storage: kodev1alpha1.KodeStorageSpec{
		// 				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		// 				Resources: corev1.VolumeResourceRequirements{
		// 					Requests: corev1.ResourceList{
		// 						corev1.ResourceStorage: resource.MustParse("1Gi"),
		// 					},
		// 				},
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, kode)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("updating the KodeTemplate resource")
		// 	updatedImage := "lscr.io/linuxserver/code-server:latest-updated"
		// 	kodeTemplate.Spec.SharedKodeTemplateSpec.Image = updatedImage
		// 	err = k8sClient.Update(ctx, kodeTemplate)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("checking if the Deployment has been updated")
		// 	deployment := &appsv1.Deployment{}
		// 	Eventually(func() error {
		// 		return k8sClient.Get(ctx, typeNamespacedName, deployment)
		// 	}, timeout, interval).Should(Succeed())

		// 	Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(updatedImage))
		// })

		// It("should reconcile the Kode resource when the EnvoyProxyConfig is updated", func() {
		// 	By("creating the custom resource for the Kind KodeTemplate")
		// 	kodeTemplate := &kodev1alpha1.KodeTemplate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      kodeTemplateName,
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.KodeTemplateSpec{
		// 			SharedKodeTemplateSpec: kodev1alpha1.SharedKodeTemplateSpec{
		// 				EnvoyProxyRef: kodev1alpha1.EnvoyProxyReference{
		// 					Kind: envoyProxyConfigKind,
		// 					Name: envoyProxyConfigName,
		// 				},
		// 				Image: kodeTemplateImage,
		// 				Type:  "code-server",
		// 			},
		// 		},
		// 	}
		// 	err := k8sClient.Create(ctx, kodeTemplate)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("creating the custom resource for the Kind EnvoyProxyConfig")
		// 	envoyProxyConfig := &kodev1alpha1.EnvoyProxyConfig{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      envoyProxyConfigName,
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.EnvoyProxyConfigSpec{
		// 			SharedEnvoyProxyConfigSpec: kodev1alpha1.SharedEnvoyProxyConfigSpec{
		// 				Image: envoyProxyConfigImage,
		// 				HTTPFilters: []kodev1alpha1.HTTPFilter{{
		// 					Name:        "filter1",
		// 					TypedConfig: runtime.RawExtension{Raw: []byte(envoyProxyConfigFilter)},
		// 				}},
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, envoyProxyConfig)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("creating the custom resource for the Kind Kode")
		// 	kode := &kodev1alpha1.Kode{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      kodeResourceName,
		// 			Namespace: resourceNamespace,
		// 		},
		// 		Spec: kodev1alpha1.KodeSpec{
		// 			TemplateRef: kodev1alpha1.KodeTemplateReference{
		// 				Kind:      kodeTemplateKind,
		// 				Name:      kodeTemplateName,
		// 				Namespace: resourceNamespace,
		// 			},
		// 			Storage: kodev1alpha1.KodeStorageSpec{
		// 				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		// 				Resources: corev1.VolumeResourceRequirements{
		// 					Requests: corev1.ResourceList{
		// 						corev1.ResourceStorage: resource.MustParse("1Gi"),
		// 					},
		// 				},
		// 			},
		// 		},
		// 	}
		// 	err = k8sClient.Create(ctx, kode)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("updating the EnvoyProxyConfig resource")
		// 	updatedImage := "envoyproxy/envoy:v1.31-latest"
		// 	envoyProxyConfig.Spec.SharedEnvoyProxyConfigSpec.Image = updatedImage
		// 	err = k8sClient.Update(ctx, envoyProxyConfig)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	By("checking if the Deployment has been updated")
		// 	deployment := &appsv1.Deployment{}
		// 	Eventually(func() error {
		// 		return k8sClient.Get(ctx, typeNamespacedName, deployment)
		// 	}, timeout, interval).Should(Succeed())

		// 	// Check if the Envoy sidecar container image is updated
		// 	found := false
		// 	for _, container := range deployment.Spec.Template.Spec.Containers {
		// 		if container.Image == updatedImage {
		// 			found = true
		// 			break
		// 		}
		// 	}
		// 	Expect(found).To(BeTrue())
		// })
	})
})
