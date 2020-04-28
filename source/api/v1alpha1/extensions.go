package v1alpha1

import "k8s.io/apimachinery/pkg/types"

/////////////
/// NAMES ///
/////////////

// NamespacedName returns the namespaced name of the zone.
func (z DNSZoneRef) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: "",
		Name:      z.Name,
	}
}

// NamespacedName returns the namespaced name of the secret.
func (s SecretRef) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: s.Namespace,
		Name:      s.Name,
	}
}

// NamespacedName returns the namespaced name of the service.
func (s ServiceIPSource) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: s.Namespace,
		Name:      s.Name,
	}
}

// CertificateNamespacedName returns the namespaced name of a potential TLS certificate definition.
func (r DNSRecord) CertificateNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name: func() string {
			if r.Spec.TLS == nil {
				return ""
			}
			return r.Spec.TLS.CertificateName
		}(),
		Namespace: r.Namespace,
	}
}

////////////////
/// EQUALITY ///
////////////////

// Equal returns whether the objects are equal.
func (status DNSZoneStatus) Equal(other DNSZoneStatus) bool {
	return status.Domain == other.Domain
}

// Equal returns whether the objects are equal.
func (status DNSResourceStatus) Equal(other DNSResourceStatus) bool {
	return status.Ready == other.Ready
}

/////////////
/// EMPTY ///
/////////////

// Empty returns whether all optional fields are nil.
func (source IPSource) Empty() bool {
	return source.Static == nil && source.Service == nil && source.Node == nil
}
