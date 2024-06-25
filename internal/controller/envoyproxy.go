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
	"crypto/sha1"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
)

// TODO: Improve how CUE files are loaded and used in the code.

//go:embed cue/bootstrap_schema.cue
var embeddedCueSchemaFile string

//go:embed cue/bootstrap.cue
var embeddedBootstrapCueFile string

const (
	RestartPolicyAlways     corev1.ContainerRestartPolicy = corev1.ContainerRestartPolicyAlways
	EnvoyProxyContainerName                               = "envoy-proxy"
	EnvoyProxyRunAsUser                                   = 1111
	ProxyInitContainerName                                = "proxy-init"
	ProxyInitContainerImage                               = "openpolicyagent/proxy_init:v8"
	BasicAuthContainerPort                                = 9001
	BasicAuthContainerImage                               = "ghcr.io/emil-jacero/grpc-basic-auth:latest"
	BasicAuthContainerName                                = "basic-auth-service"
	RouterFilterType                                      = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	ExtAuthzFilterType                                    = "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz"
	BasicAuthFilterType                                   = "type.googleapis.com/envoy.extensions.filters.http.basic_auth.v3.BasicAuth"
)

func writeEmbeddedFilesToTempDir() (string, error) {
	tempDir := os.TempDir()

	files := map[string]string{
		"bootstrap_schema.cue": embeddedCueSchemaFile,
		"bootstrap.cue":        embeddedBootstrapCueFile,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write file %s", name)
		}
	}

	return tempDir, nil
}

// BootstrapConfigOptions contains the options for rendering the bootstrap config.
type BootstrapConfigOptions struct {
	HTTPFilters []kodev1alpha1.HTTPFilter
	Clusters    []kodev1alpha1.Cluster
	ExposePort  int32
}

// Define the Router filter
var RouterHTTPFilter = kodev1alpha1.HTTPFilter{
	Name: "envoy.filters.http.router",
	TypedConfig: runtime.RawExtension{
		Raw: []byte(fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType)),
	},
}

// Define the BasicAuth filter
var BasicAuthHTTPFilter = kodev1alpha1.HTTPFilter{
	Name: "envoy.filters.http.ext_authz.basic_auth",
	TypedConfig: runtime.RawExtension{
		Raw: []byte(fmt.Sprintf(`{"@type": "%s",
			"with_request_body": {
				"max_request_bytes": 8192,
				"allow_partial_message": true
			},
			"failure_mode_allow": false,
			"grpc_service": {
				"envoy_grpc": {
					"cluster_name": "basic-auth-service"
				},
				"timeout": "0.250s"
			},
			"transport_api_version": "V3"
		}}`, ExtAuthzFilterType)),
	},
}

// Define the BasicAuth cluster
var BasicAuthCluster = kodev1alpha1.Cluster{
	Name:           BasicAuthContainerName,
	ConnectTimeout: "0.25s",
	Type:           "STRICT_DNS",
	LbPolicy:       "ROUND_ROBIN",
	TypedExtensionProtocolOptions: runtime.RawExtension{
		Raw: []byte(fmt.Sprintf(`{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": {
			"@type": "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions",
			"explicit_http_config": {
				"http2_protocol_options": {},
			},
		}`)),
	},
	LoadAssignment: kodev1alpha1.LoadAssignment{
		ClusterName: BasicAuthContainerName,
		Endpoints: []kodev1alpha1.Endpoints{{
			LbEndpoints: []kodev1alpha1.LbEndpoint{{
				Endpoint: kodev1alpha1.Endpoint{
					Address: kodev1alpha1.Address{
						SocketAddress: kodev1alpha1.SocketAddress{
							Address:   BasicAuthContainerName,
							PortValue: BasicAuthContainerPort,
						},
					},
				},
			}},
		}},
	},
}

func EnsureRouterFilter(filters []kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	for i, filter := range filters {
		if string(filter.TypedConfig.Raw) == fmt.Sprintf(`{"@type": "%s"}`, RouterFilterType) {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	return append(filters, RouterHTTPFilter)
}

func unifyAndValidate(schema, value cue.Value) (cue.Value, error) {
	unifiedValue := schema.Unify(value)
	if err := unifiedValue.Validate(); err != nil {
		return cue.Value{}, errors.Wrap(err, "failed to validate unified value")
	}
	return unifiedValue, nil
}

func encodeToYAML(value cue.Value) (string, error) {
	yamlBytes, err := yaml.Encode(value)
	if err != nil {
		return "", errors.Wrap(err, "failed to encode YAML")
	}
	return string(yamlBytes), nil
}

// encodeAndFillPath encodes the provided data and fills it into the CUE value at the specified path.
// It takes a CUE context, a CUE value, the CUE parse path, the CUE value path, the CUE schema file, and the data as input.
// It returns the updated CUE value and an error if any.
func encodeAndFillPath(ctx *cue.Context, value cue.Value, cueParsePath string, cueValuePath string, cueSchemaFile string, data interface{}) (cue.Value, error) {
	schema := ctx.CompileString(cueSchemaFile).LookupPath(cue.ParsePath(cueParsePath))
	if schema.Err() != nil {
		return value, errors.Wrap(schema.Err(), "Failed to parse path")
	}

	valueAsCUE := ctx.Encode(data)
	if valueAsCUE.Err() != nil {
		return value, errors.Wrap(valueAsCUE.Err(), "Failed to encode data")
	}

	unifiedValue, err := unifyAndValidate(schema, valueAsCUE)
	if err != nil {
		return value, err
	}

	// Fill the unified value into the CUE value at the specified path
	value = value.FillPath(cue.ParsePath(cueValuePath), unifiedValue)
	return value, nil
}

// getBootstrapConfig generates the bootstrap configuration based on the provided options.
// It reads the CUE schema file, builds a CUE instance, encodes and fills the required paths,
// and finally encodes the resulting value to YAML format.
// The generated bootstrap configuration is returned as a string.
// If any error occurs during the process, it is returned along with an empty string.
func getBootstrapConfig(log logr.Logger, options BootstrapConfigOptions) (string, error) {
	log.Info("Starting getBootstrapConfig", "options", options)

	ctx := cuecontext.New()

	// Ensure that the Router filter is included in the HTTP filters
	options.HTTPFilters = EnsureRouterFilter(options.HTTPFilters)

	// Write the embedded files to a temporary directory
	tempDir, err := writeEmbeddedFilesToTempDir()
	if err != nil {
		log.Error(err, "Failed to write embedded files to temp directory")
		return "", errors.Wrap(err, "failed to write embedded files to temp directory")
	}
	defer os.RemoveAll(tempDir)

	// Load the CUE instance from the temporary directory
	cueFiles := []string{
		filepath.Join(tempDir, "bootstrap_schema.cue"),
		filepath.Join(tempDir, "bootstrap.cue"),
	}
	inst := load.Instances(cueFiles, nil)[0]
	if inst.Err != nil {
		log.Error(inst.Err, "Failed to load CUE instance")
		return "", errors.Wrap(inst.Err, "failed to load CUE instance")
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		log.Error(value.Err(), "Failed to build CUE instance")
		return "", errors.Wrap(value.Err(), "failed to build CUE instance")
	}

	type encodeAndFillPathArgs struct {
		cueParsePath  string
		cueValuePath  string
		cueSchemaFile string
		data          interface{}
	}

	args := []encodeAndFillPathArgs{
		{"#HTTPFilters", "#GoHttpFilters", embeddedCueSchemaFile, options.HTTPFilters},
		{"#Clusters", "#GoClusters", embeddedCueSchemaFile, options.Clusters},
		{"#Port", "#GoLocalServicePort", embeddedCueSchemaFile, uint32(InternalServicePort)},
		{"#Port", "#GoExposePort", embeddedCueSchemaFile, uint32(options.ExposePort)},
	}

	for _, arg := range args {
		log.Info("Encoding and filling path", "path", arg.cueParsePath, "data", arg.data)
		var err error
		value, err = encodeAndFillPath(ctx, value, arg.cueParsePath, arg.cueValuePath, arg.cueSchemaFile, arg.data)
		if err != nil {
			log.Error(err, "Failed to encode and fill path", "path", arg.cueParsePath)
			return "", err
		}
	}

	result, err := encodeToYAML(value)
	if err != nil {
		log.Error(err, "Failed to encode to YAML")
		return "", err
	}

	log.Info("Successfully rendered bootstrap config")
	return result, nil
}

// constructEnvoyProxyContainer constructs the Envoy Proxy container and the Init container.
// It takes a logger, the sharedKodeTemplateSpec, and the sharedEnvoyProxyTemplateSpec as input.
// It returns the constructed Envoy Proxy container, the Init container, and an error if any.
func constructEnvoyProxyContainer(upstreamLog logr.Logger,
	templateSpec *kodev1alpha1.SharedKodeTemplateSpec,
	envoyProxyConfigSpec *kodev1alpha1.SharedEnvoyProxyConfigSpec,
	username string,
	password string) (corev1.Container, []corev1.Container, error) {
	var log logr.Logger
	log = upstreamLog.WithName("constructEnvoyProxyContainer")

	var initContainers []corev1.Container

	log.Info("Constructing Envoy Proxy container", "templateSpec", templateSpec, "envoyProxyConfigSpec", envoyProxyConfigSpec)

	exposePort := templateSpec.Port

	// Construct the Proxy Init container
	proxySetupContainer := corev1.Container{
		Name:  ProxyInitContainerName,
		Image: ProxyInitContainerImage,
		// Documentation: https://github.com/open-policy-agent/opa-envoy-plugin/blob/main/proxy_init/proxy_init.sh
		Args: []string{"-p", strconv.Itoa(int(exposePort)), "-u", strconv.FormatInt(*int64Ptr(EnvoyProxyRunAsUser), 16)},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsNonRoot: boolPtr(false),
			RunAsUser:    int64Ptr(0),
		},
	}
	initContainers = append(initContainers, proxySetupContainer)

	// Check if AuthType is set to "basic" and password is provided
	if envoyProxyConfigSpec.AuthType == "basic" && (password != "") {
		BasicAuthHTTPFilter := BasicAuthHTTPFilter
		BasicAuthCluster := BasicAuthCluster
		basicAuthContainer := corev1.Container{
			Name:  BasicAuthContainerName,
			Image: BasicAuthContainerImage,
			Ports: []corev1.ContainerPort{
				{Name: "grpc", ContainerPort: BasicAuthContainerPort},
			},
			Env: []corev1.EnvVar{
				{Name: "AUTH_USERNAME", Value: username},
				{Name: "AUTH_PASSWORD", Value: password},
			},
			// RestartPolicy: corev1.ContainerRestartPolicyAlways,
		}
		initContainers = append(initContainers, basicAuthContainer)
		envoyProxyConfigSpec.HTTPFilters = prependFilter(envoyProxyConfigSpec.HTTPFilters, BasicAuthHTTPFilter)
		envoyProxyConfigSpec.Clusters = append(envoyProxyConfigSpec.Clusters, BasicAuthCluster)
		log.Info("Added Basic Auth service container", "username", username)
	}

	config, err := getBootstrapConfig(log, BootstrapConfigOptions{
		HTTPFilters: envoyProxyConfigSpec.HTTPFilters,
		Clusters:    envoyProxyConfigSpec.Clusters,
		ExposePort:  exposePort,
	})
	if err != nil {
		log.Error(err, "Failed to get bootstrap config")
		return corev1.Container{}, []corev1.Container{}, err
	}

	envoyContainer := corev1.Container{
		Name:  EnvoyProxyContainerName,
		Image: envoyProxyConfigSpec.Image,
		Args: []string{
			"--config-yaml", config,
		},
		Ports: []corev1.ContainerPort{
			{Name: "envoy-http", ContainerPort: exposePort},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: int64Ptr(EnvoyProxyRunAsUser),
		},
	}

	log.Info("Successfully constructed Envoy Proxy and Init containers")
	return envoyContainer, initContainers, nil
}

// hashPassword hashes the password using SHA-1 and base64 encoding to be compatible with htpasswd SHA
func hashPassword(password string) string {
	passwordHash := sha1.Sum([]byte(password))
	return "{SHA}" + base64.StdEncoding.EncodeToString(passwordHash[:])
}

// prependFilter prepends the filter to the list of HTTP filters
func prependFilter(filters []kodev1alpha1.HTTPFilter, filter kodev1alpha1.HTTPFilter) []kodev1alpha1.HTTPFilter {
	return append([]kodev1alpha1.HTTPFilter{filter}, filters...)
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
