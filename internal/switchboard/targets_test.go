package switchboard

import (
	"context"
	"testing"
	"time"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestServiceTargetTargets(t *testing.T) {
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
	targets, err := target.Targets(ctx, ctrlClient)
	require.Nil(t, err)
	assert.ElementsMatch(t, service.Spec.ClusterIPs, targets)

	// Update the service to provide a load balancer
	service.Spec.Type = "LoadBalancer"
	err = ctrlClient.Update(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the load balancer IP
	time.Sleep(time.Second)
	name := client.ObjectKeyFromObject(&service)
	err = ctrlClient.Get(ctx, name, &service)
	require.Nil(t, err)
	targets, err = target.Targets(ctx, ctrlClient)
	require.Nil(t, err)
	assert.ElementsMatch(t, []string{service.Status.LoadBalancer.Ingress[0].IP}, targets)
}

func TestServiceTargetTargetsFromService(t *testing.T) {
	var target serviceTarget

	service := v1.Service{
		Spec: v1.ServiceSpec{ClusterIPs: []string{"10.0.0.5"}},
	}

	// Source cluster IP
	targets := target.targetsFromService(service)
	assert.ElementsMatch(t, service.Spec.ClusterIPs, targets)

	// Source multiple cluster IPs
	service.Spec.ClusterIPs = []string{"10.0.0.5", "2001:db8::1"}
	targets = target.targetsFromService(service)
	assert.ElementsMatch(t, service.Spec.ClusterIPs, targets)

	// Source IP from status
	service.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{
		IP: "192.168.5.5",
	}}
	targets = target.targetsFromService(service)
	assert.ElementsMatch(t, []string{"192.168.5.5"}, targets)

	// Source hostname from status
	service.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{
		Hostname: "example.lb.identifier.amazonaws.com",
	}}
	targets = target.targetsFromService(service)
	assert.ElementsMatch(t, []string{"example.lb.identifier.amazonaws.com"}, targets)

	// Ensure hostname takes precedence
	service.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{
		IP:       "192.168.5.5",
		Hostname: "example.lb.identifier.amazonaws.com",
	}}
	targets = target.targetsFromService(service)
	assert.ElementsMatch(t, []string{"example.lb.identifier.amazonaws.com"}, targets)

	// Ensure only one hostname
	service.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{{
		Hostname: "example.lb.identifier.amazonaws.com",
	}, {
		Hostname: "example2.lb.identifier.amazonaws.com",
	}}
	targets = target.targetsFromService(service)
	assert.ElementsMatch(t, []string{"example.lb.identifier.amazonaws.com"}, targets)
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
	ips, err := target.Targets(ctx, nil)
	require.Nil(t, err)
	assert.ElementsMatch(t, expectedIPs, ips)
}
