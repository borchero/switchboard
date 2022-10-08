package switchboard

import "strings"

const (
	ingressAnnotationKey = "kubernetes.io/ingress.class"
	ignoreAnnotationKey  = "switchboard.borchero.com/ignore"
)

// Selector allows to easily determine if a resource with a set of annotations should be processed.
type Selector struct {
	ingressClass *string
}

// NewSelector creates a new selector which selects resources with the
// `kubernetes.io/ingress.class` set to the specified value if it is not `nil`.
func NewSelector(ingressClass *string) Selector {
	return Selector{ingressClass}
}

// Matches determines whether the provided set of annotation match the selector. If the method
// returns `true`, the resource from which the annotations were obtained should be processed.
func (s Selector) Matches(annotations map[string]string) bool {
	// If the ignore annotation is set, selector never matches
	if ignore, ok := annotations[ignoreAnnotationKey]; ok {
		if ignore == "true" || ignore == "all" {
			return false
		}
	}

	// If the selector has an associated ingress class, the ingress class must match
	if s.ingressClass != nil {
		if ingressClass, ok := annotations[ingressAnnotationKey]; ok {
			return *s.ingressClass == ingressClass
		}
		// No ingress class present
		return false
	}

	// Otherwise, any ingress class is fine
	return true
}

// MatchesIntegration returns whether the provided set of annotations match the provided
// integration.
func (s Selector) MatchesIntegration(annotations map[string]string, integration string) bool {
	if ignore, ok := annotations[ignoreAnnotationKey]; ok {
		if ignore == "true" || ignore == "all" {
			return false
		}
		// Iterate over list of values set for `ignore` annotation
		for _, ignored := range strings.Split(ignore, ",") {
			if strings.TrimSpace(ignored) == integration {
				return false
			}
		}
	}
	return true
}
