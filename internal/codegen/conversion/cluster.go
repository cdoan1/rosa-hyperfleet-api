package conversion

import v2alpha1 "github.com/openshift/rosa-regional-platform-api/api/v2alpha1"

// ClusterServiceSetFields holds platform-injected values for cluster creation.
type ClusterServiceSetFields struct {
	CloudURL   string
	Placement  string
	CreatorARN string
}

// InjectClusterServiceSet merges service-set fields into a typed cluster spec.
// Only non-empty values are injected. Placement is only set if not already
// present in the spec (allowing client-provided values to take precedence).
func InjectClusterServiceSet(spec *v2alpha1.ClusterSpec, ssf ClusterServiceSetFields) {
	if ssf.CloudURL != "" {
		spec.CloudUrl = ssf.CloudURL
	}
	if ssf.Placement != "" && spec.Placement == "" {
		spec.Placement = ssf.Placement
	}
	if ssf.CreatorARN != "" {
		spec.CreatorARN = ssf.CreatorARN
	}
}

// RewriteCloudURLWithID sets CloudUrl to baseURL/clusterID in a response spec.
func RewriteCloudURLWithID(spec *v2alpha1.ClusterSpec, baseURL, clusterID string) {
	spec.CloudUrl = baseURL + "/" + clusterID
}
