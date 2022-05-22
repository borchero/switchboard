package k8s

import (
	"context"
	"testing"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteIfFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)
	namespace, shutdown := k8tests.NewNamespace(ctx, t, client)
	defer shutdown()

	// Create a service
	service := k8tests.DummyService("my-service", namespace, 80)
	err := client.Create(ctx, &service)
	require.Nil(t, err)

	// Multiple deletes should not result in an error
	err = DeleteIfFound(ctx, client, &service)
	assert.Nil(t, err)

	err = DeleteIfFound(ctx, client, &service)
	assert.Nil(t, err)
}
