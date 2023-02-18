package integrations

import (
	"context"
	"fmt"
	"testing"

	"github.com/borchero/switchboard/internal/k8tests"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCertManagerUpdateResource(t *testing.T) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, client)
	defer shutdown()

	// Create a dummy service as owner
	owner := k8tests.DummyService("my-service", namespace, 80)
	err := client.Create(ctx, &owner)
	require.Nil(t, err)
	integration := NewCertManager(client, certmanager.Certificate{
		Spec: certmanager.CertificateSpec{
			IssuerRef: cmmeta.ObjectReference{
				Kind: "ClusterIssuer",
				Name: "my-issuer",
			},
		},
	})

	// Nothing should be created if no hosts or no tls is set
	tlsName := "test-tls"

	var info IngressInfo

	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	assert.Len(t, getCertificates(ctx, t, client, namespace), 0)

	info = IngressInfo{TLSSecretName: &tlsName}
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	assert.Len(t, getCertificates(ctx, t, client, namespace), 0)

	info = IngressInfo{Hosts: []string{"example.com"}}
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	assert.Len(t, getCertificates(ctx, t, client, namespace), 0)

	// If both are set, we should see a certificate created
	info = IngressInfo{Hosts: []string{"example.com"}, TLSSecretName: &tlsName}
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)

	certificates := getCertificates(ctx, t, client, namespace)
	assert.Len(t, certificates, 1)
	assert.Equal(t, fmt.Sprintf("%s-tls", owner.Name), certificates[0].Name)
	assert.Equal(t, tlsName, certificates[0].Spec.SecretName)
	assert.Equal(t, "ClusterIssuer", certificates[0].Spec.IssuerRef.Kind)
	assert.Equal(t, "my-issuer", certificates[0].Spec.IssuerRef.Name)
	assert.ElementsMatch(t, info.Hosts, certificates[0].Spec.DNSNames)

	// We should see an update if we change any info
	info.Hosts = []string{"example.com", "www.example.com"}
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	certificates = getCertificates(ctx, t, client, namespace)
	assert.Len(t, certificates, 1)
	assert.Equal(t, tlsName, certificates[0].Spec.SecretName)
	assert.ElementsMatch(t, info.Hosts, certificates[0].Spec.DNSNames)

	updatedTLSName := "new-test-tls"
	info.TLSSecretName = &updatedTLSName
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	certificates = getCertificates(ctx, t, client, namespace)
	assert.Len(t, certificates, 1)
	assert.Equal(t, updatedTLSName, certificates[0].Spec.SecretName)
	assert.ElementsMatch(t, info.Hosts, certificates[0].Spec.DNSNames)

	// When no hosts are set, the certificate should be removed again
	info.Hosts = nil
	err = integration.UpdateResource(ctx, &owner, info)
	require.Nil(t, err)
	assert.Len(t, getCertificates(ctx, t, client, namespace), 0)
}

//-------------------------------------------------------------------------------------------------
// UTILS
//-------------------------------------------------------------------------------------------------

func getCertificates(
	ctx context.Context, t *testing.T, ctrlClient client.Client, namespace string,
) []certmanager.Certificate {
	var list certmanager.CertificateList
	err := ctrlClient.List(ctx, &list, &client.ListOptions{
		Namespace: namespace,
	})
	require.Nil(t, err)
	return list.Items
}
