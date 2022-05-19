package k8s

import (
	"context"
	"testing"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpsert(t *testing.T) {
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
			Selector: map[string]string{
				"app.kubernetes.io/name": "notfound",
			},
			Ports: []v1.ServicePort{{
				Port: 80,
				Name: "http",
			}},
		},
	}
	op, err := Upsert(ctx, client, &service)
	require.Nil(t, err)
	assert.Equal(t, OperationCreated, op)
	assert.Equal(t, int32(80), service.Spec.Ports[0].Port)

	// Update the service with a new port
	service.Spec.Ports[0].Port = 8080
	op, err = Upsert(ctx, client, &service)
	require.Nil(t, err)
	assert.Equal(t, OperationUpdated, op)
	assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)
}
