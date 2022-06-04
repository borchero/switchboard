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

func TestServiceTargetIP(t *testing.T) {
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
	target := NewServiceTarget(service.Name, service.Namespace)
	ips, err := target.IPs(ctx, ctrlClient)
	require.Nil(t, err)
	assert.ElementsMatch(t, service.Spec.ClusterIPs, ips)

	// Update the service to provide a load balancer
	service.Spec.Type = "LoadBalancer"
	err = ctrlClient.Update(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the load balancer IP
	time.Sleep(time.Second)
	name := client.ObjectKeyFromObject(&service)
	err = ctrlClient.Get(ctx, name, &service)
	require.Nil(t, err)
	ips, err = target.IPs(ctx, ctrlClient)
	require.Nil(t, err)
	assert.ElementsMatch(t, []string{service.Status.LoadBalancer.Ingress[0].IP}, ips)
}

func TestServiceTargetNamespacedName(t *testing.T) {
	target := NewServiceTarget("my-service", "my-namespace")
	name := target.NamespacedName()
	assert.Equal(t, "my-service", name.Name)
	assert.Equal(t, "my-namespace", name.Namespace)
}

func TestStaticTargetIPs(t *testing.T) {
	ctx := context.Background()
	expectedIPs := []string{"127.0.0.1", "2001:db8::1"}
	target := NewStaticTarget(expectedIPs...)
	ips, err := target.IPs(ctx, nil)
	require.Nil(t, err)
	assert.ElementsMatch(t, expectedIPs, ips)
}
