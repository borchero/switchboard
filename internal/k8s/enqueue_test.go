package k8s

import (
	"context"
	"log/slog"
	"testing"

	"github.com/borchero/switchboard/internal/ext"
	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEnqueueMapFunc(t *testing.T) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	ctrlClient := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, ctrlClient)
	defer shutdown()

	// Create a couple of services
	service1 := k8tests.DummyService("my-service-1", namespace, 80)
	err := ctrlClient.Create(ctx, &service1)
	require.Nil(t, err)
	service2 := k8tests.DummyService("my-service-2", namespace, 80)
	err = ctrlClient.Create(ctx, &service2)
	require.Nil(t, err)
	service3 := k8tests.DummyService("my-service-3", namespace, 80)
	err = ctrlClient.Create(ctx, &service3)
	require.Nil(t, err)

	// Create the enqueue function which is triggered by service 1
	var services v1.ServiceList
	enqueuer := EnqueueMapFunc(
		ctrlClient, slog.Default(), &service1, &services,
		func(list *v1.ServiceList) []client.Object {
			return ext.Map(list.Items, func(v v1.Service) client.Object { return &v })
		},
	)

	// Check whether enqueue only happens for service1
	assert.Greater(t, len(enqueuer(ctx, &service1)), 0)
	assert.Len(t, enqueuer(ctx, &service2), 0)
	assert.Len(t, enqueuer(ctx, &service3), 0)

	// Check whether distinct services are returned for enqueue
	names := []string{"my-service-1", "my-service-2", "my-service-3"}
	var found []string

	for _, obj := range enqueuer(ctx, &service1) {
		if obj.Namespace == namespace {
			found = append(found, obj.Name)
		}
	}
	assert.ElementsMatch(t, names, found)
}
