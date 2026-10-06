package k8tests

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	externaldnsv1alpha1 "sigs.k8s.io/external-dns/apis/v1alpha1"
)

func TestExternalDNSRequestOptions(t *testing.T) {
	_, err := runtime.NewParameterCodec(NewScheme()).EncodeParameters(&metav1.GetOptions{}, externaldnsv1alpha1.GroupVersion)
	if err != nil {
		t.Fatalf("encode external-dns request options: %v", err)
	}
}
