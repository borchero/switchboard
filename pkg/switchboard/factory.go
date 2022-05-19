package switchboard

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/pkg/k8s"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/external-dns/endpoint"
)

const (
	managedByLabelKey = "kubernetes.io/managed-by"
)

// Factory is a type which simplifies the creation of Kubernetes resources.
type Factory struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewFactory returns a factory to create Kubernetes resources.
func NewFactory(client client.Client, scheme *runtime.Scheme) *Factory {
	return &Factory{client, scheme}
}

//-------------------------------------------------------------------------------------------------
// DNSEndpoint
//-------------------------------------------------------------------------------------------------

func (f Factory) createDNSEndpoint(
	owner metav1.Object, endpoints []*endpoint.Endpoint,
) (endpoint.DNSEndpoint, error) {
	// Copy annotations if required
	annotations := map[string]string{}
	if ingressClass, ok := owner.GetAnnotations()[ingressAnnotationKey]; ok {
		annotations[ingressAnnotationKey] = ingressClass
	}

	// Create type
	dnsEndpoint := endpoint.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:        owner.GetName(),
			Namespace:   owner.GetNamespace(),
			Annotations: annotations,
			Labels: map[string]string{
				managedByLabelKey: "switchboard",
			},
		},
		Spec: endpoint.DNSEndpointSpec{
			Endpoints: endpoints,
		},
	}

	// Set reference
	if err := ctrl.SetControllerReference(owner, &dnsEndpoint, f.scheme); err != nil {
		return dnsEndpoint, fmt.Errorf(
			"failed to set controller reference for DNS endpoint: %w", err,
		)
	}
	return dnsEndpoint, nil
}

// UpsertDNSEndpoint creates a new DNS endpoint resource (or updates it if it exists), copies the
// ingress annotation from the provided resource and sets the provided resource as owner.
func (f Factory) UpsertDNSEndpoint(
	ctx context.Context, owner metav1.Object, endpoints []*endpoint.Endpoint,
) error {
	dnsEndpoint, err := f.createDNSEndpoint(owner, endpoints)
	if err != nil {
		return fmt.Errorf("failed to create DNS endpoint: %w", err)
	}

	// Upsert resource
	if _, err := k8s.Upsert(ctx, f.client, &dnsEndpoint); err != nil {
		return fmt.Errorf("failed to upsert DNS endpoint: %w", err)
	}
	return nil
}

//-------------------------------------------------------------------------------------------------
// Certificate
//-------------------------------------------------------------------------------------------------

func (f *Factory) createCertificate(
	owner metav1.Object, issuer cmmeta.ObjectReference, secretName string, hosts []string,
) (certmanager.Certificate, error) {
	// Copy annotations if required
	annotations := map[string]string{}
	if ingressClass, ok := owner.GetAnnotations()[ingressAnnotationKey]; ok {
		annotations[ingressAnnotationKey] = ingressClass
	}

	certificate := certmanager.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tls", owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				managedByLabelKey: "switchboard",
			},
			Annotations: annotations,
		},
		Spec: certmanager.CertificateSpec{
			SecretName: secretName,
			DNSNames:   hosts,
			IssuerRef:  issuer,
		},
	}

	if err := ctrl.SetControllerReference(owner, &certificate, f.scheme); err != nil {
		return certificate, fmt.Errorf(
			"failed to set controller reference for TLS certificate: %w", err,
		)
	}
	return certificate, nil
}

// UpsertCertificate creates a new certificate resource (or updates it if it exists), copies the
// ingress annotation from the provided resource and sets the provided resource as owner.
func (f Factory) UpsertCertificate(
	ctx context.Context,
	owner metav1.Object,
	issuer cmmeta.ObjectReference,
	secretName string,
	hosts []string,
) error {
	certificate, err := f.createCertificate(owner, issuer, secretName, hosts)
	if err != nil {
		return fmt.Errorf("failed to create TLS certificate: %w", err)
	}

	// Upsert resource
	if _, err := k8s.Upsert(ctx, f.client, &certificate); err != nil {
		return fmt.Errorf("failed to upsert TLS certificate: %w", err)
	}
	return nil
}
