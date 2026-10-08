package v1alpha1

import hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

// ClusterCapabilitiesSpec mirrors HyperShift v1beta1 Capabilities while
// exposing its two immutable capability lists through the HyperFleet API.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Capabilities is immutable. Changes might result in unpredictable and disruptive behavior."
// +kubebuilder:validation:XValidation:rule="has(self.enabled) && has(self.disabled) ? self.enabled.all(e, !(e in self.disabled)) : true",message="Capabilities cannot be both enabled and disabled at once."
type ClusterCapabilitiesSpec struct {
	// enabled explicitly enables the specified capabilities on the hosted cluster.
	// Once set, this field cannot be changed.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=25
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Enabled is immutable. Changes might result in unpredictable and disruptive behavior."
	// +optional
	Enabled []hypershiftv1beta1.OptionalCapability `json:"enabled,omitempty"`

	// disabled explicitly disables the specified capabilities on the hosted cluster.
	// Once set, this field cannot be changed.
	// +k8s:openapi-gen=true
	// +hyperfleet:write-mode=immutable
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=25
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Disabled is immutable. Changes might result in unpredictable and disruptive behavior."
	// +kubebuilder:validation:XValidation:rule="!self.exists(cap, cap == 'Ingress') || self.exists(cap, cap == 'Console')",message="Ingress capability can only be disabled if Console capability is also disabled."
	// +optional
	Disabled []hypershiftv1beta1.OptionalCapability `json:"disabled,omitempty"`
}
