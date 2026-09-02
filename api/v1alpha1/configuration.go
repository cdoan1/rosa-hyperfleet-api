package v1alpha1

import (
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterConfiguration specifies configuration for individual OCP components in the cluster.
// This is a HyperFleet-owned mirror of hypershiftv1beta1.ClusterConfiguration that allows
// us to add granular markers to nested fields like kubelet config.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.ClusterConfiguration
type ClusterConfiguration struct {
	// authentication contains configuration for the cluster authentication.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	Authentication *ClusterAuthentication `json:"authentication,omitempty"`

	// featureGate contains the desired configuration for feature gates.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	FeatureGate *FeatureGateConfiguration `json:"featureGate,omitempty"`

	// image contains the configuration for internal registry.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	Image *ImageConfiguration `json:"image,omitempty"`

	// ingress contains the configuration for ingress.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	Ingress *IngressConfiguration `json:"ingress,omitempty"`

	// network contains the configuration for cluster networking.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=service-set
	Network *NetworkConfiguration `json:"network,omitempty"`

	// oauth contains the configuration for OAuth.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	OAuth *OAuthConfiguration `json:"oauth,omitempty"`

	// scheduler contains the configuration for scheduler.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	Scheduler *SchedulerConfiguration `json:"scheduler,omitempty"`

	// proxy contains the configuration for the cluster-wide proxy.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	Proxy *ProxyConfiguration `json:"proxy,omitempty"`

	// kubelet contains the configuration for kubelet on nodes.
	// +hyperfleet:write-mode=service-set
	Kubelet *KubeletConfig `json:"kubelet,omitempty"`

	// machineConfig contains the configuration for machine-level settings.
	// +hyperfleet:write-mode=service-set
	MachineConfig *MachineConfigSpec `json:"machineConfig,omitempty"`
}

// KubeletConfig specifies kubelet configuration with granular markers for customer control.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.KubeletConfig
type KubeletConfig struct {
	// +hyperfleet:write-mode=mutable
	MaxPods *int32 `json:"maxPods,omitempty"`

	// +hyperfleet:write-mode=mutable
	PodPidsLimit *int64 `json:"podPidsLimit,omitempty"`

	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxProperties=32
	SystemReserved map[string]string `json:"systemReserved,omitempty"`

	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxProperties=32
	KubeReserved map[string]string `json:"kubeReserved,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxProperties=32
	EvictionHard map[string]string `json:"evictionHard,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxProperties=32
	EvictionSoft map[string]string `json:"evictionSoft,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxProperties=32
	EvictionSoftGracePeriod map[string]string `json:"evictionSoftGracePeriod,omitempty"`

	// +hyperfleet:write-mode=mutable
	ImageGCHighThresholdPercent *int32 `json:"imageGCHighThresholdPercent,omitempty"`

	// +hyperfleet:write-mode=mutable
	ImageGCLowThresholdPercent *int32 `json:"imageGCLowThresholdPercent,omitempty"`

	// +hyperfleet:write-mode=mutable
	ImageMinimumGCAge *metav1.Duration `json:"imageMinimumGCAge,omitempty"`

	// +openshift:enable:FeatureGate=HyperFleetKubeletAdvanced
	// +hyperfleet:write-mode=mutable
	SerializeImagePulls *bool `json:"serializeImagePulls,omitempty"`

	// +openshift:enable:FeatureGate=HyperFleetKubeletAdvanced
	// +hyperfleet:write-mode=mutable
	RegistryPullQPS *int32 `json:"registryPullQPS,omitempty"`

	// +openshift:enable:FeatureGate=HyperFleetKubeletAdvanced
	// +hyperfleet:write-mode=mutable
	RegistryBurst *int32 `json:"registryBurst,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	CPUManagerPolicy *string `json:"cpuManagerPolicy,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxProperties=32
	CPUManagerPolicyOptions map[string]string `json:"cpuManagerPolicyOptions,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	CPUManagerReconcilePeriod *metav1.Duration `json:"cpuManagerReconcilePeriod,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	TopologyManagerPolicy *string `json:"topologyManagerPolicy,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	TopologyManagerScope *string `json:"topologyManagerScope,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxItems=256
	AllowedUnsafeSysctls []string `json:"allowedUnsafeSysctls,omitempty"`

	// +hyperfleet:write-mode=mutable
	StreamingConnectionIdleTimeout *metav1.Duration `json:"streamingConnectionIdleTimeout,omitempty"`

	// +hyperfleet:write-mode=mutable
	ContainerLogMaxSize *string `json:"containerLogMaxSize,omitempty"`

	// +hyperfleet:write-mode=mutable
	ContainerLogMaxFiles *int32 `json:"containerLogMaxFiles,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	MemoryThrottlingFactor *float64 `json:"memoryThrottlingFactor,omitempty"`
}

// Placeholder types for configuration areas not yet exposed.

type ClusterAuthentication struct{}

type FeatureGateConfiguration struct{}

type ImageConfiguration struct{}

type IngressConfiguration struct{}

// NetworkConfiguration specifies cluster network configuration.
// +hyperfleet:upstream-reduced-object=configv1.NetworkSpec
type NetworkConfiguration struct {
	// clusterNetwork is the IP address pool to use for pod IPs.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=32
	ClusterNetwork []ClusterNetworkEntry `json:"clusterNetwork,omitempty"`

	// serviceNetwork is the IP address pool for services.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=32
	ServiceNetwork []string `json:"serviceNetwork,omitempty"`

	// networkType is the plugin to be deployed (e.g. OVNKubernetes).
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	NetworkType string `json:"networkType,omitempty"`

	// externalIP defines configuration for Service.ExternalIP.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	ExternalIP *configv1.ExternalIPConfig `json:"externalIP,omitempty"`

	// serviceNodePortRange is the port range for Services of type NodePort.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=mutable
	ServiceNodePortRange string `json:"serviceNodePortRange,omitempty"`

	// networkDiagnostics defines network diagnostics configuration.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	NetworkDiagnostics configv1.NetworkDiagnostics `json:"networkDiagnostics,omitempty"`
}

// ClusterNetworkEntry is a contiguous block of IP addresses for pods.
// +hyperfleet:upstream-reduced-object=configv1.ClusterNetworkEntry
type ClusterNetworkEntry struct {
	// cidr is the complete block for pod IPs.
	// +hyperfleet:write-mode=immutable
	CIDR string `json:"cidr"`

	// hostPrefix is the size (prefix) of block to allocate to each node.
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:Minimum=0
	HostPrefix uint32 `json:"hostPrefix,omitempty"`
}

type OAuthConfiguration struct{}

type SchedulerConfiguration struct{}

type ProxyConfiguration struct{}

// MachineConfigSpec specifies machine-level configuration.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.MachineConfigSpec
type MachineConfigSpec struct {
	// +openshift:enable:FeatureGate=HyperFleetMachineConfig
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=128
	AllowedKernelArguments []string `json:"allowedKernelArguments,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxItems=128
	KernelArguments []string `json:"kernelArguments,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxItems=64
	SystemdUnits []SystemdUnit `json:"systemdUnits,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxItems=256
	Files []FileSpec `json:"files,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	KernelType *string `json:"kernelType,omitempty"`

	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	// +kubebuilder:validation:MaxItems=64
	Extensions []string `json:"extensions,omitempty"`
}

type SystemdUnit struct {
	Name string `json:"name"`
	Enabled  *bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:MaxLength=65536
	Contents string `json:"contents,omitempty"`
	// +kubebuilder:validation:MaxItems=16
	Dropins []SystemdDropin `json:"dropins,omitempty"`
}

type SystemdDropin struct {
	Name string `json:"name"`
	// +kubebuilder:validation:MaxLength=32768
	Contents string `json:"contents,omitempty"`
}

type FileSpec struct {
	Path string `json:"path"`
	// +kubebuilder:validation:MaxLength=262144
	Contents  string  `json:"contents,omitempty"`
	Mode      *int32  `json:"mode,omitempty"`
	User      *string `json:"user,omitempty"`
	Group     *string `json:"group,omitempty"`
	Overwrite *bool   `json:"overwrite,omitempty"`
}
