package controllers

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	traefik "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v3/pkg/types"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	externaldnsv1alpha1 "sigs.k8s.io/external-dns/apis/v1alpha1"
)

func TestSimpleIngressTCP(t *testing.T) {
	runTestTCP(t, testCaseTCP{
		Ingress: traefik.IngressRouteTCP{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress-tcp",
			},
			Spec: traefik.IngressRouteTCPSpec{
				Routes: []traefik.RouteTCP{{
					Match: "HostSNI(`www.example.com`)",
				}},
				TLS: &traefik.TLSTCP{
					SecretName: "www-tls-certificate",
				},
			},
		},
		DNSNames: []string{"www.example.com"},
	})
}

func TestIngressTCPNoTLS(t *testing.T) {
	runTestTCP(t, testCaseTCP{
		Ingress: traefik.IngressRouteTCP{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress-tcp",
			},
			Spec: traefik.IngressRouteTCPSpec{
				Routes: []traefik.RouteTCP{{
					Match: "HostSNI(`www.example.com`)",
				}},
			},
		},
		DNSNames: []string{"www.example.com"},
	})
}

func TestIngressTCPWildcard(t *testing.T) {
	runTestTCP(t, testCaseTCP{
		Ingress: traefik.IngressRouteTCP{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress-tcp",
			},
			Spec: traefik.IngressRouteTCPSpec{
				Routes: []traefik.RouteTCP{{
					Match: "HostSNI(`*`)",
				}},
			},
		},
		DNSNames: []string{},
	})
}

func TestIngressTCPCustomDNS(t *testing.T) {
	runTestTCP(t, testCaseTCP{
		Ingress: traefik.IngressRouteTCP{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress-tcp",
			},
			Spec: traefik.IngressRouteTCPSpec{
				Routes: []traefik.RouteTCP{{
					Match: "HostSNI(`example.com`)",
				}},
				TLS: &traefik.TLSTCP{
					SecretName: "www-tls-certificate",
					Domains: []traefiktypes.Domain{{
						Main: "example.net",
						SANs: []string{
							"*.example.net",
						},
					}},
				},
			},
		},
		DNSNames: []string{"example.net", "*.example.net"},
	})
}

func TestIngressTCPMultipleRules(t *testing.T) {
	runTestTCP(t, testCaseTCP{
		Ingress: traefik.IngressRouteTCP{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress-tcp",
			},
			Spec: traefik.IngressRouteTCPSpec{
				Routes: []traefik.RouteTCP{{
					Match: "HostSNI(`example.com`, `www.example.com`)",
				}, {
					Match: "HostSNI(`v2.example.com`)",
				}},
				TLS: &traefik.TLSTCP{
					SecretName: "www-tls-certificate",
				},
			},
		},
		DNSNames: []string{"example.com", "www.example.com", "v2.example.com"},
	})
}

//-------------------------------------------------------------------------------------------------
// TESTING UTILITIES
//-------------------------------------------------------------------------------------------------

type testCaseTCP struct {
	Ingress  traefik.IngressRouteTCP
	DNSNames []string
}

func runTestTCP(t *testing.T, test testCaseTCP) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, client)
	defer shutdown()

	// Create objects and run reconciliation
	service := k8tests.DummyService("traefik", namespace, 80)
	err := client.Create(ctx, &service)
	require.Nil(t, err)

	test.Ingress.Namespace = namespace
	err = client.Create(ctx, &test.Ingress)
	require.Nil(t, err)

	config := createConfigTCP(&service)
	runReconciliationTCP(ctx, t, client, test.Ingress, config)

	// Check whether the outputs are valid
	// 1) Certificate
	certificateName := types.NamespacedName{
		Name:      fmt.Sprintf("%s-tls", test.Ingress.Name),
		Namespace: namespace,
	}
	var certificate certmanager.Certificate
	err = client.Get(ctx, certificateName, &certificate)
	if test.Ingress.Spec.TLS == nil {
		assert.True(t, apierrors.IsNotFound(err))
	} else {
		assert.Nil(t, err)
		assert.ElementsMatch(t, test.DNSNames, certificate.Spec.DNSNames)
		assert.Equal(t,
			config.Integrations.CertManager.Template.Spec.IssuerRef.Kind,
			certificate.Spec.IssuerRef.Kind,
		)
		assert.Equal(t,
			config.Integrations.CertManager.Template.Spec.IssuerRef.Name,
			certificate.Spec.IssuerRef.Name,
		)
		assert.Equal(t, test.Ingress.Spec.TLS.SecretName, certificate.Spec.SecretName)
	}

	// 2) DNS records
	endpointName := types.NamespacedName{Name: test.Ingress.Name, Namespace: namespace}
	var dnsEndpoint externaldnsv1alpha1.DNSEndpoint
	err = client.Get(ctx, endpointName, &dnsEndpoint)
	if len(test.DNSNames) == 0 {
		assert.True(t, apierrors.IsNotFound(err))
	} else {
		assert.Nil(t, err)
		assert.Len(t, dnsEndpoint.Spec.Endpoints, len(test.DNSNames))
		for _, ep := range dnsEndpoint.Spec.Endpoints {
			assert.Len(t, ep.Targets, 1)
			assert.Equal(t, service.Spec.ClusterIP, ep.Targets[0])
		}
	}
}

//-------------------------------------------------------------------------------------------------
// OBJECT CREATION
//-------------------------------------------------------------------------------------------------

func runReconciliationTCP(
	ctx context.Context,
	t *testing.T,
	client client.Client,
	ingress traefik.IngressRouteTCP,
	config configv1.Config,
) {
	reconciler, err := NewIngressRouteTCPReconciler(client, slog.Default(), config)
	require.Nil(t, err)
	_, err = reconciler.Reconcile(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace},
	})
	require.Nil(t, err)
}

func createConfigTCP(service *v1.Service) configv1.Config {
	return configv1.Config{
		Integrations: configv1.IntegrationConfigs{
			ExternalDNS: &configv1.ExternalDNSIntegrationConfig{
				TargetService: &configv1.ServiceRef{
					Name:      service.Name,
					Namespace: service.Namespace,
				},
			},
			CertManager: &configv1.CertManagerIntegrationConfig{
				Template: certmanager.Certificate{
					Spec: certmanager.CertificateSpec{
						IssuerRef: cmmeta.IssuerReference{
							Kind: "ClusterIssuer",
							Name: "my-issuer",
						},
					},
				},
			},
		},
	}
}
