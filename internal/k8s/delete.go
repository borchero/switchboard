package k8s

import (
	"context"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteIfFound deletes the given resource from the cluster and returns an error only if the
// deletion fails. If the resource does not exist, no error will be returned.
func DeleteIfFound(ctx context.Context, client client.Client, resource client.Object) error {
	if err := client.Delete(ctx, resource); err != nil {
		if apierrs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete existing resource: %w", err)
	}
	return nil
}
