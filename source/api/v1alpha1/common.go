package v1alpha1

/////////////////
/// CONSTANTS ///
/////////////////

// ServiceIPType describes a type of service IP.
// +kubebuilder:validation:Enum=ExternalIP;ClusterIP
type ServiceIPType string

// NodeIPType describes a type of a node IP.
// +kubebuilder:validation:Enum=ExternalIP;InternalIP
type NodeIPType string

// IssuerKind describes the kind an issuer may be.
// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
type IssuerKind string

// TimeToLive describes the time for which a DNS record should be kept in the cache.
// +kubebuilder:validation:Minimum=60
// +kubebuilder:validation:Maximum=86400
// +kubebuilder:validation:ExclusiveMaximum=false
type TimeToLive int

const (
	// ServiceIPTypeExternal describes the use of a service's external IP.
	ServiceIPTypeExternal = ServiceIPType("ExternalIP")

	// ServiceIPTypeCluster describes the use of a service's cluster IP.
	ServiceIPTypeCluster = ServiceIPType("ClusterIP")

	// NodeIPTypeExternal describes the use of a node's external IP address.
	NodeIPTypeExternal = NodeIPType("ExternalIP")

	// NodeIPTypeInternal describes the use of a node's internal IP address.
	NodeIPTypeInternal = NodeIPType("InternalIP")

	// IssuerKindNamespaced describes a namespaced issuer.
	IssuerKindNamespaced = IssuerKind("Issuer")

	// IssuerKindCluster describes a cluster-wide issuer.
	IssuerKindCluster = IssuerKind("ClusterIssuer")
)

//////////////
/// RECORD ///
//////////////

// DNSRecordHosts defines the specification for DNS zones.
type DNSRecordHosts struct {
	// +kubebuilder:validation:MinItems=1
	Hosts  []string `json:"hosts"`
	Cnames []string `json:"cnames,omitempty"`
}

/////////////////
/// IP SOURCE ///
/////////////////

// IPSource represents the source of an IP to set.
type IPSource struct {
	Static  *StaticIPSource  `json:"staticIP,omitempty"`
	Service *ServiceIPSource `json:"serviceIP,omitempty"`
	Node    *NodeIPSource    `json:"nodeIP,omitempty"`
}

// StaticIPSource refers to a static IP.
type StaticIPSource struct {
	IP string `json:"ip"`
}

// ServiceIPSource refers the source of an IP to the IP of a service, either public or private.
type ServiceIPSource struct {
	Name      string        `json:"name"`
	Namespace string        `json:"namespace,omitempty"`
	Type      ServiceIPType `json:"type,omitempty"`
}

// NodeIPSource refers to the IP of a random node.
type NodeIPSource struct {
	LabelSelectors map[string]string `json:"matchLabels,omitempty"`
	Type           NodeIPType        `json:"type,omitempty"`
}

////////////
/// CORE ///
////////////

// SecretRef references a Kubernetes secret in the same or another namespace.
type SecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}
