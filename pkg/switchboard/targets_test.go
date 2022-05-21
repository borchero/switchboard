package switchboard

import (
	"context"
	"testing"
	"time"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIP(t *testing.T) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	ctrlClient := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, ctrlClient)
	defer shutdown()

	// Create a new service
	service := k8tests.DummyService("my-service", namespace, 80)
	err := ctrlClient.Create(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the cluster IP
	target := NewTarget(service.Name, service.Namespace)
	ip, err := target.IP(ctx, ctrlClient)
	require.Nil(t, err)
	assert.Equal(t, service.Spec.ClusterIP, ip)

	// Update the service to provide a load balancer
	service.Spec.Type = "LoadBalancer"
	err = ctrlClient.Update(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the load balancer IP
	time.Sleep(time.Second)
	name := client.ObjectKeyFromObject(&service)
	err = ctrlClient.Get(ctx, name, &service)
	require.Nil(t, err)
	ip, err = target.IP(ctx, ctrlClient)
	require.Nil(t, err)
	assert.Equal(t, service.Status.LoadBalancer.Ingress[0].IP, ip)
}

func TestNamespacedName(t *testing.T) {
	target := NewTarget("my-service", "my-namespace")
	name := target.NamespacedName()
	assert.Equal(t, "my-service", name.Name)
	assert.Equal(t, "my-namespace", name.Namespace)
}
