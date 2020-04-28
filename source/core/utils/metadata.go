package utils

import (
	"math/rand"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	rollmeAnnotation = "switchboard.borchero.com/rollme"
	finalizerName    = "finalizer.switchboard.borchero.com"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// UpdateRollmeAnnotation adds/updates a random "rollme" annotation which can be leveraged to force
// the update of a resource.
func UpdateRollmeAnnotation(meta *metav1.ObjectMeta) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[rollmeAnnotation] = strconv.Itoa(rand.Int())
}

// AttachFinalizerIfNeeded adds a finalizer with a predefined name to the given metadata object if
// it is not present yet. Returns whether the finalizer has actually been added.
func AttachFinalizerIfNeeded(meta *metav1.ObjectMeta) bool {
	if !contains(meta.Finalizers, finalizerName) {
		meta.Finalizers = append(meta.Finalizers, finalizerName)
		return true
	}
	return false
}

// DetachFinalizerIfNeeded removes a finalizer with a predefined name from the given metadata
// object if it is present. Returns whether the finalizer has actually been removed.
func DetachFinalizerIfNeeded(meta *metav1.ObjectMeta) bool {
	newFinalizers, ret := remove(meta.Finalizers, finalizerName)
	meta.Finalizers = newFinalizers
	return ret
}

func contains(list []string, val string) bool {
	for _, item := range list {
		if val == item {
			return true
		}
	}
	return false
}

func remove(list []string, val string) ([]string, bool) {
	result := make([]string, 0)
	removed := false
	for _, item := range list {
		if item == val {
			removed = true
			continue
		}
		result = append(result, item)
	}
	return result, removed
}
