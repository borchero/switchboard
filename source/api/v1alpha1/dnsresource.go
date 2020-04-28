package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSType describes the set of valid types for a DNS resource.
// +kubebuilder:validation:Enum=A;CNAME
type DNSType string

const (
	// DNSTypeA describes a DNS A resource.
	DNSTypeA = DNSType("A")

	// DNSTypeCname describes a DNS CNAME resource.
	DNSTypeCname = DNSType("CNAME")
)

// DNSResource represents a DNS resource record, i.e. a single entry in a particular DNS zone.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TYPE",type=string,JSONPath=.spec.type
// +kubebuilder:printcolumn:name="DOMAIN",type=string,JSONPath=.spec.domain
// +kubebuilder:printcolumn:name="DATA",type=string,JSONPath=.spec.data
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=.status.ready
type DNSResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSResourceSpec   `json:"spec"`
	Status DNSResourceStatus `json:"status,omitempty"`
}

// DNSResourceSpec represents the specification for a DNSResource.
type DNSResourceSpec struct {
	ZoneName string     `json:"zoneName"`
	Domain   string     `json:"domain"`
	Type     DNSType    `json:"type"`
	Data     string     `json:"data"`
	TTL      TimeToLive `json:"ttl"`
}

// DNSResourceStatus indicates whether the item has been added to its backend successfully.
type DNSResourceStatus struct {
	Ready bool `json:"ready"`
}

// DNSResourceList represents multiple DNSResource CRDs.
// +kubebuilder:object:root=true
type DNSResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSResource `json:"items"`
}
