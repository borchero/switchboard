package switchboard

import (
	"context"
	"testing"
	"time"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestIP(t *testing.T) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, client)
	defer shutdown()

	// Create a new service
	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-service",
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Type: "ClusterIP",
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

	// Check whether we find the cluster IP
	target := NewTarget(service.Name, service.Namespace)
	ip, err := target.IP(ctx, client)
	require.Nil(t, err)
	assert.Equal(t, service.Spec.ClusterIP, ip)

	// Update the service to provide a load balancer
	service.Spec.Type = "LoadBalancer"
	err = client.Update(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the load balancer IP
	time.Sleep(time.Second)
	name := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
	err = client.Get(ctx, name, &service)
	require.Nil(t, err)
	ip, err = target.IP(ctx, client)
	require.Nil(t, err)
	assert.Equal(t, service.Status.LoadBalancer.Ingress[0].IP, ip)
}
