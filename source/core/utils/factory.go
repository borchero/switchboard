package utils

import (
	"context"
	"fmt"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/backends"
	"go.borchero.com/typewriter"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BackendFactory provides a principled way of creating backends from specifications.
type BackendFactory interface {
	Create(spec v1alpha1.DNSZoneSpec) (backends.DNSZone, error)
}

type backendFactoryImpl struct {
	client client.Client
	log    typewriter.Logger
}

// NewBackendFactory creates a new backend factory that talks to the Kubernetes API via the
// specified client and provides logging with the specified logger.
func NewBackendFactory(client client.Client, log typewriter.Logger) BackendFactory {
	return &backendFactoryImpl{client, log}
}

func (factory *backendFactoryImpl) Create(spec v1alpha1.DNSZoneSpec) (backends.DNSZone, error) {
	ctx := context.Background()

	var backend backends.DNSZone
	var err error

	switch {
	case spec.CloudDNS != nil:
		factory.log.Info("creating CloudDNS backend")
		backend, err = factory.newCloudDNSBackend(ctx, spec)
	default:
		err = fmt.Errorf("Zone specification does not define a baceknd")
	}

	if err != nil {
		return nil, err
	}

	factory.log.
		WithV(typewriter.KV("domain", backend.Domain())).
		Info("Successfully created backend")

	return backend, nil
}

func (factory *backendFactoryImpl) newCloudDNSBackend(
	ctx context.Context, spec v1alpha1.DNSZoneSpec,
) (backends.DNSZone, error) {
	// 1) Get credentials
	cloudDNS := spec.CloudDNS
	credentials, err := factory.readSecretRef(ctx, cloudDNS.CredentialsSecret)
	if err != nil {
		return nil, fmt.Errorf("CloudDNS credentials cannot be loaded: %s", err)
	}

	// 2)
	backend, err := backends.NewCloudDNSZone(ctx, cloudDNS.ZoneName, credentials)
	if err != nil {
		return nil, fmt.Errorf("Error initializing CloudDNS backend: %s", err)
	}

	return backend, nil
}

func (factory *backendFactoryImpl) readSecretRef(
	ctx context.Context, ref v1alpha1.SecretRef,
) ([]byte, error) {
	// 1) Get secret
	var secret v1.Secret
	err := factory.client.Get(ctx, ref.NamespacedName(), &secret)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving secret '%s': %s", ref.NamespacedName(), err)
	}

	// 2) Get path
	credentials, ok := secret.Data[ref.Key]
	if !ok {
		return nil, fmt.Errorf("Secret path '%s' not found", ref.Key)
	}

	return credentials, nil
}
