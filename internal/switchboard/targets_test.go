package switchboard

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/borchero/zeus/pkg/zeus"
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
	target := NewServiceTarget(service.Name, service.Namespace, zeus.Logger(ctx))
	targets, err := target.Targets(ctx, ctrlClient, nil)
	require.Nil(t, err)
	assert.ElementsMatch(t, service.Spec.ClusterIPs, targets)

	// Update the service to provide a load balancer
	service.Spec.Type = "LoadBalancer"
	err = ctrlClient.Update(ctx, &service)
	require.Nil(t, err)

	// Check whether we find the load balancer IP
	for i := 0; i < 15 && len(service.Status.LoadBalancer.Ingress) == 0; i++ {
		time.Sleep(time.Second) // Wait for service to be ready
		// time.Sleep(time.Second)
		name := client.ObjectKeyFromObject(&service)
		err = ctrlClient.Get(ctx, name, &service)
		require.Nil(t, err)
	}

	targets, err = target.Targets(ctx, ctrlClient, nil)
	require.Nil(t, err)
	var external string
	if len(service.Status.LoadBalancer.Ingress[0].IP) > 0 {
		external = service.Status.LoadBalancer.Ingress[0].IP
	}
	if len(service.Status.LoadBalancer.Ingress[0].Hostname) > 0 {
		external = service.Status.LoadBalancer.Ingress[0].Hostname
	}
	assert.ElementsMatch(t, []string{external}, targets)

	explictTarget := fmt.Sprintf("1.1.1.1, 1::1 , www.test.bla, %s/my-service, my-service, 2.2.2.2, 2::2 , www.test.blub,  ,", namespace)
	_, err = target.Targets(ctx, ctrlClient, &explictTarget)
	require.NotNil(t, err)

	explictTarget = fmt.Sprintf("www.test.bla, %s/my-service, my-service,  ", namespace)
	_, err = target.Targets(ctx, ctrlClient, &explictTarget)
	require.NotNil(t, err)

	explictTarget = fmt.Sprintf("1.1.1.1, 1::1, %s/my-service, my-service,  ", namespace)
	_, err = target.Targets(ctx, ctrlClient, &explictTarget)
	require.NotNil(t, err)

	explictTarget = fmt.Sprintf("%s/my-service, my-service,  ", namespace)
	ips, err := target.Targets(ctx, ctrlClient, &explictTarget)
	require.Nil(t, err)
	assert.ElementsMatch(t, ips, []string{external})

	explictTarget = "www.test.bla,  "
	ips, err = target.Targets(ctx, ctrlClient, &explictTarget)
	require.Nil(t, err)
	assert.ElementsMatch(t, ips, []string{"www.test.bla"})

	explictTarget = "2::2, 1.1.1.1, 1.1.1.1, 1::1, 1::1, "
	ips, err = target.Targets(ctx, ctrlClient, &explictTarget)
	require.Nil(t, err)
	assert.ElementsMatch(t, ips, []string{"1.1.1.1", "1::1", "2::2"})
}

func TestServiceTargetTargetsFromService(t *testing.T) {
	target := serviceTarget{}
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
	target := NewServiceTarget("my-service", "my-namespace", zeus.Logger(context.Background()))
	name := target.NamespacedName()
	assert.Equal(t, "my-service", name.Name)
	assert.Equal(t, "my-namespace", name.Namespace)
}

func TestStaticTargetIPs(t *testing.T) {
	ctx := context.Background()
	expectedIPs := []string{"127.0.0.1", "2001:db8::1"}
	target := NewStaticTarget(expectedIPs...)
	ips, err := target.Targets(ctx, nil, nil)
	require.Nil(t, err)
	assert.ElementsMatch(t, expectedIPs, ips)
}
