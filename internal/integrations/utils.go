package integrations

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func reconcileMetadata(source metav1.Object, target metav1.Object, scheme *runtime.Scheme) error {
	// Reconcile labels
	labels := defaultEmpty(target.GetLabels())
	labels[managedByLabelKey] = "switchboard"
	target.SetLabels(labels)

	// Reconcile annotations
	annotations := defaultEmpty(target.GetAnnotations())
	if ingressClass, ok := source.GetAnnotations()[ingressAnnotationKey]; ok {
		annotations[ingressAnnotationKey] = ingressClass
	} else {
		delete(annotations, ingressAnnotationKey)
	}
	target.SetAnnotations(annotations)

	// Set controller reference
	if err := ctrl.SetControllerReference(source, target, scheme); err != nil {
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
