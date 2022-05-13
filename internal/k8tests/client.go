package k8tests

import (
	"os"
	"path/filepath"
	"testing"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/require"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/external-dns/endpoint"
)

// NewScheme returns a newly configured scheme which registers all types that are relevant for
// Switchboard.
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	// >>> core types
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// >>> cert-manager
	utilruntime.Must(certmanager.AddToScheme(scheme))
	// >>> traefik
	utilruntime.Must(traefik.AddToScheme(scheme))
	// >>> external-dns
	groupVersion := schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}
	scheme.AddKnownTypes(groupVersion,
		&endpoint.DNSEndpoint{},
		&endpoint.DNSEndpointList{},
	)
	metav1.AddToGroupVersion(scheme, groupVersion)
	return scheme
}

// NewClient returns a new Kubernetes client from the configuration available at ~/.kube/config.
// The test fails if initialization fails.
func NewClient(t *testing.T, scheme *runtime.Scheme) client.Client {
	configPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	require.Nil(t, err)
	client, err := client.New(config, client.Options{Scheme: scheme})
	require.Nil(t, err)
	return client
}
