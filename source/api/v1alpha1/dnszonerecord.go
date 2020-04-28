package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DNSZoneRecord models a DNSRecord in a particular zone.
// +kubebuilder:object:root=true
type DNSZoneRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec DNSZoneRecordSpec `json:"spec"`
}

// DNSZoneRecordSpec represents the desired state of a DNSZoneRecord. It only references the zone's
// name to pass it down to the DNSResource which makes use of the zone's backend.
type DNSZoneRecordSpec struct {
	DNSRecordHosts `json:",inline"`
	IPSource       `json:",inline"`
	ZoneName       string     `json:"zoneName"`
	TTL            TimeToLive `json:"ttl"`
}

// DNSZoneRecordList represents multiple DNSZoneRecord CRDs.
// +kubebuilder:object:root=true
type DNSZoneRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSZoneRecord `json:"items"`
}
