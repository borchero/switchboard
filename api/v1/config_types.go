package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

func init() {
	SchemeBuilder.Register(&Config{})
}

//+kubebuilder:object:root=true

// Config is the Schema for the configs API
type Config struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	IngressConfig IngressSet `json:"ingressConfig"`
}

// IngressSet represents the configuration for a set of ingress resources.
type IngressSet struct {
	TargetService ServiceRef           `json:"targetService"`
	Issuer        CertificateIssuerRef `json:"certificateIssuer"`
	Selector      IngressSelector      `json:"selector,omitempty"`
}

// IngressSelector can be used to limit operations to ingresses with a specific class.
type IngressSelector struct {
	IngressClass *string `json:"ingressClass,omitempty"`
}

// ServiceRef uniquely describes a Kubernetes service.
type ServiceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// CertificateIssuerRef uniquely describes a certificate issuer in Kubernetes.
type CertificateIssuerRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
