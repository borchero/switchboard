package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSRecord represents the DNSRecord CRD which maps a set of hosts and cnames to a DNSZone.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec DNSRecordSpec `json:"spec"`
}

// DNSRecordSpec defines the specification for a DNSRecord CRD.
type DNSRecordSpec struct {
	DNSRecordHosts `json:",inline"`
	// +kubebuilder:validation:MinItems=1
	Zones []DNSZoneRef `json:"zones"`
	TTL   *TimeToLive  `json:"ttl,omitempty"`
	TLS   *TLSSpec     `json:"tls,omitempty"`
}

// DNSZoneRef references a DNS zone and optionally overrides the default IP source.
type DNSZoneRef struct {
	IPSource `json:",inline"`
	Name     string      `json:"name"`
	TTL      *TimeToLive `json:"ttl,omitempty"`
}

// TLSSpec defines a specification to obtain a TLS certificate for a set of hostnames.
type TLSSpec struct {
	CertificateName string    `json:"certificateName"`
	SecretName      string    `json:"secretName,omitempty"`
	Issuer          IssuerRef `json:"issuer"`
}

// IssuerRef is used to reference a cert-manager issuer.
type IssuerRef struct {
	Name string     `json:"name"`
	Kind IssuerKind `json:"kind,omitempty"`
}

// DNSRecordList represent multiple DNSRecord CRDs.
// +kubebuilder:object:root=true
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSRecord `json:"items"`
}
