package mapper

// OperatorIAMRole represents an operator IAM role from ROSA CLI
type OperatorIAMRole struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	RoleARN   string `json:"role_arn"`
}

// AWSRolesRef represents the rolesRef structure for HostedCluster CR
type AWSRolesRef struct {
	NetworkARN              string `json:"networkARN"`
	StorageARN              string `json:"storageARN"`
	ImageRegistryARN        string `json:"imageRegistryARN"`
	KubeCloudControllerARN  string `json:"kubeCloudControllerARN"`
	NodePoolManagementARN   string `json:"nodePoolManagementARN"`
	ControlPlaneOperatorARN string `json:"controlPlaneOperatorARN"`
	IngressARN              string `json:"ingressARN"`
}

// CloudProviderConfig represents the cloudProviderConfig for HostedCluster CR
type CloudProviderConfig struct {
	SubnetID string `json:"subnetID"`
	VpcID    string `json:"vpcID"`
	Zone     string `json:"zone"`
}
