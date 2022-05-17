package controllers

import (
	"context"
	"fmt"
	"testing"

	configv1 "github.com/borchero/switchboard/api/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefiktypes "github.com/traefik/traefik/v2/pkg/types"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	service := createService(ctx, t, client, namespace)
	test.Ingress.Namespace = namespace
	err := client.Create(ctx, &test.Ingress)
	require.Nil(t, err)
	reconciler := runReconciliation(ctx, t, client, scheme, test.Ingress, service)

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
		assert.Equal(t, reconciler.IngressConfig.Issuer.Kind, certificate.Spec.IssuerRef.Kind)
		assert.Equal(t, reconciler.IngressConfig.Issuer.Name, certificate.Spec.IssuerRef.Name)
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
	scheme *runtime.Scheme,
	ingress traefik.IngressRoute,
	service *v1.Service,
) *IngressRouteReconciler {
	reconciler := IngressRouteReconciler{
		Client: client,
		Scheme: scheme,
		Logger: zap.NewNop(),
		IngressConfig: configv1.IngressSet{
			TargetService: configv1.ServiceRef{
				Name:      service.Name,
				Namespace: service.Namespace,
			},
			Issuer: configv1.CertificateIssuerRef{
				Kind: "ClusterIssuer",
				Name: "issuer",
			},
		},
	}
	_, err := reconciler.Reconcile(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace},
	})
	require.Nil(t, err)
	return &reconciler
}

func createService(
	ctx context.Context, t *testing.T, client client.Client, namespace string,
) *v1.Service {
	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traefik",
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": "notfound",
			},
			Ports: []v1.ServicePort{{
				Port: 80,
				Name: "http",
			}},
		},
	}
	err := client.Create(ctx, &service)
	require.Nil(t, err)
	return &service
}
