package controllers

import (
	"context"
	"fmt"
	"testing"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v2/pkg/types"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestSimpleIngress(t *testing.T) {
	runTest(t, testCase{
		Ingress: traefik.IngressRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress",
			},
			Spec: traefik.IngressRouteSpec{
				Routes: []traefik.Route{{
					Kind:  "Rule",
					Match: "Host(`www.example.com`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}},
				TLS: &traefik.TLS{
					SecretName: "www-tls-certificate",
				},
			},
		},
		DNSNames: []string{"www.example.com"},
	})
}

func TestIngressNoTLS(t *testing.T) {
	runTest(t, testCase{
		Ingress: traefik.IngressRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress",
			},
			Spec: traefik.IngressRouteSpec{
				Routes: []traefik.Route{{
					Kind:  "Rule",
					Match: "Host(`www.example.com`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}},
			},
		},
		DNSNames: []string{"www.example.com"},
	})
}

func TestIngressNoTLSNoDNS(t *testing.T) {
	runTest(t, testCase{
		Ingress: traefik.IngressRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress",
			},
			Spec: traefik.IngressRouteSpec{
				Routes: []traefik.Route{{
					Kind:  "Rule",
					Match: "Prefix(`/test`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}},
			},
		},
		DNSNames: []string{},
	})
}

func TestIngressCustomDNS(t *testing.T) {
	runTest(t, testCase{
		Ingress: traefik.IngressRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress",
			},
			Spec: traefik.IngressRouteSpec{
				Routes: []traefik.Route{{
					Kind:  "Rule",
					Match: "Host(`example.com`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}},
				TLS: &traefik.TLS{
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

func TestIngressMultipleRules(t *testing.T) {
	runTest(t, testCase{
		Ingress: traefik.IngressRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ingress",
			},
			Spec: traefik.IngressRouteSpec{
				Routes: []traefik.Route{{
					Kind:  "Rule",
					Match: "Host(`example.com`, `www.example.com`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}, {
					Kind:  "Rule",
					Match: "Host(`v2.example.com`)",
					Services: []traefik.Service{{
						LoadBalancerSpec: traefik.LoadBalancerSpec{
							Name: "nginx",
						},
					}},
				}},
				TLS: &traefik.TLS{
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

type testCase struct {
	Ingress  traefik.IngressRoute
	DNSNames []string
}

func runTest(t *testing.T, test testCase) {
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

	config := createConfig(&service)
	runReconciliation(ctx, t, client, test.Ingress, config)

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
	var endpoint endpoint.DNSEndpoint
	err = client.Get(ctx, endpointName, &endpoint)
	if len(test.DNSNames) == 0 {
		assert.True(t, apierrors.IsNotFound(err))
	} else {
		assert.Nil(t, err)
		assert.Len(t, endpoint.Spec.Endpoints, len(test.DNSNames))
		for _, ep := range endpoint.Spec.Endpoints {
			assert.Len(t, ep.Targets, 1)
			assert.Equal(t, service.Spec.ClusterIP, ep.Targets[0])
		}
	}
}

//-------------------------------------------------------------------------------------------------
// OBJECT CREATION
//-------------------------------------------------------------------------------------------------

func runReconciliation(
	ctx context.Context,
	t *testing.T,
	client client.Client,
	ingress traefik.IngressRoute,
	config configv1.Config,
) {
	reconciler, err := NewIngressRouteReconciler(client, zap.NewNop(), config)
	require.Nil(t, err)
	_, err = reconciler.Reconcile(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace},
	})
	require.Nil(t, err)
}

func createConfig(service *v1.Service) configv1.Config {
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
						IssuerRef: cmmeta.ObjectReference{
							Kind: "ClusterIssuer",
							Name: "my-issuer",
						},
					},
				},
			},
		},
	}
}
