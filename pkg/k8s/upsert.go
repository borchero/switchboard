package k8s

import (
	"context"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Operation indicates the type of operation performed by an upsert.
type Operation string

const (
	// OperationCreated indicates that an upsert created the resource.
	OperationCreated Operation = "created"
	// OperationUpdated indicates that an upsert updated the resource.
	OperationUpdated Operation = "updated"
)

// Upsertable describes a Kubernetes resource which can be created or updated.
type Upsertable[T any] interface {
	client.Object
	DeepCopy() T
	DeepCopyInto(T)
}

// Upsert creates a new resource with the given specification if it does not yet exist in the
// cluster and updates the existing resource otherwise.
func Upsert[R Upsertable[R]](
	ctx context.Context, client client.Client, resource R,
) (Operation, error) {
	key := types.NamespacedName{
		Name:      resource.GetName(),
		Namespace: resource.GetNamespace(),
	}

	new := resource.DeepCopy() // keep around to copy into the received resource
	if err := client.Get(ctx, key, resource); err != nil {
		if apierrs.IsNotFound(err) {
			if err := client.Create(ctx, resource); err != nil {
				return Operation(""), fmt.Errorf("failed to create resource: %w", err)
			}
			return OperationCreated, nil
		}
		return Operation(""), fmt.Errorf("failed to query for existing resource: %w", err)
	}

	new.DeepCopyInto(resource) // copy into received resource to upsert
	if err := client.Update(ctx, resource); err != nil {
		return Operation(""), fmt.Errorf("failed to update resource: %w", err)
	}
	return OperationUpdated, nil
}
