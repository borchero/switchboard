package controllers

import (
	"context"
	"testing"

	configv1 "github.com/borchero/switchboard/api/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestResourceCreation(t *testing.T) {
	// Setup
	ctx := context.Background()

	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)

	namespace, shutdown := k8tests.NewNamespace(ctx, t, client)
	defer shutdown()

	// Create objects
	createService(ctx, t, client, namespace)
	createIngress(ctx, t, client, namespace)

	// Run the reconciler
	reconciler := IngressRouteReconciler{
		Client: client,
		Scheme: scheme,
		Logger: zap.NewNop(),
		IngressConfig: configv1.IngressSet{
			TargetService: configv1.ServiceRef{
				Name:      "traefik",
				Namespace: namespace,
			},
			Issuer: configv1.CertificateIssuerRef{
				Kind: "ClusterIssuer",
				Name: "issuer",
			},
		},
	}
	_, err := reconciler.Reconcile(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "my-ingress", Namespace: namespace},
	})
	require.Nil(t, err)

	// Check whether the created resources are valid.
	// Check the created certificate
	certificateName := types.NamespacedName{Name: "my-ingress-tls", Namespace: namespace}
	var certificate certmanager.Certificate
	err = client.Get(ctx, certificateName, &certificate)
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"www.example.com"}, certificate.Spec.DNSNames)
	assert.Equal(t, "ClusterIssuer", certificate.Spec.IssuerRef.Kind)
	assert.Equal(t, "issuer", certificate.Spec.IssuerRef.Name)
	assert.Equal(t, "www-tls-certificate", certificate.Spec.SecretName)

	// Check the created DNS endpoint
	endpointName := types.NamespacedName{Name: "my-ingress", Namespace: namespace}
	var endpoint endpoint.DNSEndpoint
	err = client.Get(ctx, endpointName, &endpoint)
	assert.Nil(t, err)
	assert.Len(t, endpoint.Spec.Endpoints, 1)
	assert.Len(t, endpoint.Spec.Endpoints[0].Targets, 1)

	serviceName := types.NamespacedName{Name: "traefik", Namespace: namespace}
	var service v1.Service
	err = client.Get(ctx, serviceName, &service)
	assert.Nil(t, err)
	assert.Equal(t, service.Spec.ClusterIP, endpoint.Spec.Endpoints[0].Targets[0])
}

func createService(ctx context.Context, t *testing.T, client client.Client, namespace string) {
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
}

func createIngress(ctx context.Context, t *testing.T, client client.Client, namespace string) {
	ingress := traefik.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ingress",
			Namespace: namespace,
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
	}
	err := client.Create(ctx, &ingress)
	require.Nil(t, err)
}
