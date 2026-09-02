package v1alpha1

import (
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ClusterNetworking specifies network configuration for the cluster.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.ClusterNetworking
type ClusterNetworking struct {
	// machineNetwork is the list of IP address pools for machines.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=2
	// +kubebuilder:validation:MinItems=1
	MachineNetwork []MachineNetworkEntry `json:"machineNetwork,omitempty"`

	// clusterNetwork is the list of IP address pools for pods.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=2
	// +kubebuilder:validation:MinItems=1
	ClusterNetwork []ClusterNetworkEntry `json:"clusterNetwork,omitempty"`

	// serviceNetwork is the list of IP address pools for services.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +kubebuilder:validation:MaxItems=2
	// +kubebuilder:validation:MinItems=1
	ServiceNetwork []ServiceNetworkEntry `json:"serviceNetwork,omitempty"`

	// networkType specifies the SDN provider used for cluster networking.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	NetworkType hypershiftv1beta1.NetworkType `json:"networkType,omitempty"`

	// apiServer contains advanced network settings for the API server.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	APIServer *hypershiftv1beta1.APIServerNetworking `json:"apiServer,omitempty"`

	// allocateNodeCIDRs controls whether the kube-controller-manager manages node CIDR allocation.
	// +k8s:openapi-gen=false
	// +hyperfleet:write-mode=service-set
	AllocateNodeCIDRs *hypershiftv1beta1.AllocateNodeCIDRsMode `json:"allocateNodeCIDRs,omitempty"`
}

// MachineNetworkEntry is a single IP address block for node IPs.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.MachineNetworkEntry
type MachineNetworkEntry struct {
	// cidr is the IP block address pool for machines.
	// +hyperfleet:write-mode=immutable
	CIDR string `json:"cidr"`
}

// ServiceNetworkEntry is a single IP address block for service IPs.
// +hyperfleet:upstream-reduced-object=hypershiftv1beta1.ServiceNetworkEntry
type ServiceNetworkEntry struct {
	// cidr is the IP block address pool for services.
	// +hyperfleet:write-mode=immutable
	CIDR string `json:"cidr"`
}
