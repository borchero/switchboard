package k8tests

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DummyService returns a dummy service with the specified name in the given namespace, using the
// provided target port.
func DummyService(name, namespace string, port int32) v1.Service {
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": "notfound",
			},
			Ports: []v1.ServicePort{{
				Port: port,
				Name: "http",
			}},
		},
	}
}
