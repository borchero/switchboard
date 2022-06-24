package integrations

import (
	"testing"

	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func TestReconcileMetadata(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	parent := k8tests.DummyService("my-name", "my-namespace", 80)

	// Check whether labels are set correctly
	target := k8tests.DummyService("your-name", "my-namespace", 8080)
	err := reconcileMetadata(&parent, &target, scheme)
	require.Nil(t, err)
	assert.Len(t, target.OwnerReferences, 1)
	assert.Len(t, target.Annotations, 0)
	assert.Len(t, target.Labels, 1)

	// Check whether annotations are copied correctly
	parent.Annotations = map[string]string{
		ingressAnnotationKey: "test",
		"another.annotation": "hello",
	}
	target = k8tests.DummyService("your-name", "my-namespace", 8080)
	err = reconcileMetadata(&parent, &target, scheme)
	require.Nil(t, err)
	assert.Len(t, target.OwnerReferences, 1)
	assert.Len(t, target.Annotations, 1)
	assert.Len(t, target.Labels, 1)

	// Check whether additional annotations and labels are copied
	meta := metav1.ObjectMeta{
		Labels:      map[string]string{"my-label": "my-value"},
		Annotations: map[string]string{"my-annotation-1": "1", "my-annotation-2": "2"},
	}
	target = k8tests.DummyService("your-name", "my-namespace", 8080)
	err = reconcileMetadata(&parent, &target, scheme, &meta)
	require.Nil(t, err)
	assert.Len(t, target.OwnerReferences, 1)
	assert.Len(t, target.Annotations, 3)
	assert.Len(t, target.Labels, 2)
}

func TestDefaultEmpty(t *testing.T) {
	var m1 map[string]string
	assert.Nil(t, m1)
	assert.NotNil(t, defaultEmpty(m1))
	assert.Len(t, defaultEmpty(m1), 0)

	m2 := map[string]string{"hello": "world"}
	assert.NotNil(t, m2)
	assert.NotNil(t, defaultEmpty(m2))
	assert.Len(t, defaultEmpty(m2), 1)
}
