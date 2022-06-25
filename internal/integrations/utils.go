package integrations

import (
	"fmt"

	"github.com/imdario/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func reconcileMetadata(
	owner metav1.Object, target metav1.Object, scheme *runtime.Scheme, sources ...metav1.Object,
) error {
	// Reconcile labels
	labels := defaultEmpty(target.GetLabels())
	labels[managedByLabelKey] = "switchboard"
	for _, source := range sources {
		if err := mergo.MergeWithOverwrite(&labels, source.GetLabels()); err != nil {
			return fmt.Errorf("failed to update labels: %s", err)
		}
	}
	target.SetLabels(labels)

	// Reconcile annotations
	annotations := defaultEmpty(target.GetAnnotations())
	if ingressClass, ok := owner.GetAnnotations()[ingressAnnotationKey]; ok {
		annotations[ingressAnnotationKey] = ingressClass
	} else {
		delete(annotations, ingressAnnotationKey)
	}
	for _, source := range sources {
		if err := mergo.MergeWithOverwrite(&annotations, source.GetAnnotations()); err != nil {
			return fmt.Errorf("failed to update annotations: %s", err)
		}
	}
	target.SetAnnotations(annotations)

	// Set controller reference
	if err := ctrl.SetControllerReference(owner, target, scheme); err != nil {
		return err
	}
	return nil
}

func defaultEmpty(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	return m
}
