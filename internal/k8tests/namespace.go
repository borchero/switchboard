package k8tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespace returns the (automatically generated) name of a newly created namespace along with
// a shutdown function. If creating the namespace fails, the test is aborted.
func NewNamespace(ctx context.Context, t *testing.T, client client.Client) (string, func()) {
	name := uuid.New()
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name.String(),
		},
	}
	err := client.Create(ctx, namespace)
	require.Nil(t, err)
	return name.String(), func() {
		client.Delete(ctx, namespace) // nolint:errcheck
	}
}
