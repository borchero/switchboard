package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSZone represents the DNSZone CRD.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DOMAIN",type=string,JSONPath=.status.domain
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   DNSZoneSpec   `json:"spec"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

// DNSZoneSpec defines the specification for a DNSZone CRD.
type DNSZoneSpec struct {
	RecordTemplate DNSRecordTemplate `json:"recordTemplate,omitempty"`
	CloudDNS       *CloudDNSZone     `json:"clouddns,omitempty"`
}

// DNSZoneStatus indicates the status of the zone, i.e. if it could establish a connection to the
// backend.
type DNSZoneStatus struct {
	Domain string `json:"domain"`
}

// DNSRecordTemplate can be used to supply default arguments for DNS records created for the zone.
type DNSRecordTemplate struct {
	IPSource `json:",inline"`
	TTL      *TimeToLive `json:"ttl,omitempty"`
}

// CloudDNSZone defines the metadata for a DNS zone managed by the Google Cloud Platform.
type CloudDNSZone struct {
	ZoneName          string    `json:"zoneName"`
	CredentialsSecret SecretRef `json:"credentialsSecret"`
}

// DNSZoneList represents multiple DNSZone CRDs.
// +kubebuilder:object:root=true
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSZone `json:"items"`
}
