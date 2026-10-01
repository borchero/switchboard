package main

import (
	"testing"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	externaldnsv1alpha1 "sigs.k8s.io/external-dns/apis/v1alpha1"
)

func TestExternalDNSRequestOptions(t *testing.T) {
	scheme := runtime.NewScheme()
	config := configv1.Config{}
	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{}
	initScheme(config, scheme)
	for name, scheme := range map[string]*runtime.Scheme{"controller": scheme, "tests": k8tests.NewScheme()} {
		t.Run(name, func(t *testing.T) {
			_, err := runtime.NewParameterCodec(scheme).EncodeParameters(&metav1.GetOptions{}, externaldnsv1alpha1.GroupVersion)
			if err != nil {
				t.Fatalf("encode external-dns request options: %v", err)
			}
		})
	}
}
