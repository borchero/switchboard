package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// +kubebuilder:object:generate=true
// +groupName=switchboard.borchero.com

var (
	// GroupVersion defines the group's schema.
	GroupVersion = schema.GroupVersion{
		Group:   "switchboard.borchero.com",
		Version: "v1alpha1",
	}

	// SchemeBuilder is used to add Go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{
		GroupVersion: GroupVersion,
	}

	// AddToScheme adds the types in this group to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(
		&DNSZone{}, &DNSZoneList{},
		&DNSRecord{}, &DNSRecordList{},
		&DNSZoneRecord{}, &DNSZoneRecordList{},
		&DNSResource{}, &DNSResourceList{},
	)
}
