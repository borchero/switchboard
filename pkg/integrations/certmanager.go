package integrations

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/pkg/k8s"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type certManager struct {
	client client.Client
	issuer cmmeta.ObjectReference
}

// NewCertManager initializes a new cert-manager integration which creates certificates which use
// the provided issuer.
func NewCertManager(client client.Client, issuer cmmeta.ObjectReference) Integration {
	return &certManager{client, issuer}
}

func (*certManager) Name() string {
	return "cert-manager"
}

func (*certManager) OwnedResource() client.Object {
	return &certmanager.Certificate{}
}

func (*certManager) WatchedObject() client.Object {
	return nil
}

func (c *certManager) UpdateResource(
	ctx context.Context, owner metav1.Object, info IngressInfo,
) error {
	// If the ingress does not specify a TLS secret name or specifies no hosts, no certificate
	// needs to be created.
	if info.TLSSecretName == nil || len(info.Hosts) == 0 {
		certificate := certmanager.Certificate{ObjectMeta: c.objectMeta(owner)}
		if err := k8s.DeleteIfFound(ctx, c.client, &certificate); err != nil {
			return fmt.Errorf("failed to delete TLS certificate: %w", err)
		}
		return nil
	}

	// Otherwise, we can create the certificate resource
	resource := certmanager.Certificate{ObjectMeta: c.objectMeta(owner)}
	if _, err := controllerutil.CreateOrPatch(ctx, c.client, &resource, func() error {
		// Meta
		if err := reconcileMetadata(owner, &resource, c.client.Scheme()); err != nil {
			return err
		}
		// Spec
		resource.Spec.SecretName = *info.TLSSecretName
		resource.Spec.DNSNames = info.Hosts
		resource.Spec.IssuerRef.Kind = c.issuer.Kind
		resource.Spec.IssuerRef.Name = c.issuer.Name
		return nil
	}); err != nil {
		return fmt.Errorf("failed to upsert TLS certificate: %w", err)
	}
	return nil
}

//-------------------------------------------------------------------------------------------------
// UTILS
//-------------------------------------------------------------------------------------------------

func (c *certManager) objectMeta(parent metav1.Object) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-tls", parent.GetName()),
		Namespace: parent.GetNamespace(),
	}
}
