package switchboard

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Target is a type which allows to retrieve a potentially dynamically changing IP from Kubernetes.
type Target interface {
	// Targets returns the IPv4/IPv6 addresses or hostnames that should be used as targets or an
	// error if the addresses/hostnames cannot be retrieved.
	Targets(ctx context.Context, client client.Client, target *string) ([]string, error)
	// NamespacedName returns the namespaced name of the dynamic target service or none if the IP
	// is not retrieved dynamically.
	NamespacedName() *types.NamespacedName
}

//-------------------------------------------------------------------------------------------------
// SERVICE TARGET
//-------------------------------------------------------------------------------------------------

type serviceTarget struct {
	name   types.NamespacedName
	logger *zap.Logger
}

// NewServiceTarget creates a new target which dynamically sources the IP from the provided
// Kubernetes service.
func NewServiceTarget(name, namespace string, log *zap.Logger) Target {
	return serviceTarget{
		logger: log,
		name:   types.NamespacedName{Name: name, Namespace: namespace},
	}
}

func (t serviceTarget) Targets(ctx context.Context, client client.Client, targets *string) ([]string, error) {
	// Get service
	if targets == nil {
		tmp := fmt.Sprintf("%s/%s", t.name.Namespace, t.name.Name)
		targets = &tmp
	}
	out := []string{}
	for _, target := range strings.Split(*targets, ",") {
		logger := t.logger.With(zap.String("target", target))
		target = strings.TrimSpace(target)
		if len(target) == 0 {
			continue
		}
		if net.ParseIP(target) != nil {
			logger.Debug("target is ip address")
			out = append(out, target)
			continue
		}
		// very bad regex
		if found, _ := regexp.MatchString("[0-9a-zA-Z\\-]+\\.[0-9a-zA-Z\\-]+.*", target); found {
			logger.Debug("target is cname ")
			out = append(out, target)
			continue
		}
		parts := strings.Split(target, "/")
		nsName := types.NamespacedName{
			Namespace: t.name.Namespace,
			Name:      parts[0],
		}
		if len(parts) > 1 {
			nsName.Namespace = parts[0]
			nsName.Name = parts[1]
		}
		var service v1.Service
		if err := client.Get(ctx, nsName, &service); err != nil {
			return nil, fmt.Errorf("failed to query service: %s:%s:%w", nsName.Namespace, nsName.Name, err)
		}
		logger.Debug("target is serviceaddress", zap.String("namespace", nsName.Namespace), zap.String("name", nsName.Name))
		targets := t.targetsFromService(service)
		out = append(out, targets...)
	}
	reduce := map[string]string{}
	for _, target := range out {
		class := "CNAME"
		if net.ParseIP(target) != nil {
			class = "IP"
		}
		reduce[target] = class
	}
	var itsCnameOrIp string
	countClass := 0
	out = []string{}
	for target, class := range reduce {
		out = append(out, target)
		countClass++
		if itsCnameOrIp == "" {
			itsCnameOrIp = class
		}
		if itsCnameOrIp != class {
			return nil, fmt.Errorf("cannot mix CNAME and IP addresses in target: %s", target)
		}
		if itsCnameOrIp == "CNAME" && countClass > 1 {
			return nil, fmt.Errorf("CNAME allows only one target: %s", target)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (t serviceTarget) targetsFromService(service v1.Service) []string {
	// Try to get load balancer IPs/hostnames...
	targets := make([]string, 0)
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.Hostname != "" {
			// We cannot have more than one CNAME record, the hostname overwrites everything
			targets = []string{ingress.Hostname}
			break
		}
		if ingress.IP != "" {
			targets = append(targets, ingress.IP)
		}
	}

	// ...fall back to cluster IPs
	if len(targets) == 0 {
		targets = append(targets, service.Spec.ClusterIPs...)
	}
	return targets
}

func (t serviceTarget) NamespacedName() *types.NamespacedName {
	return &t.name
}

//-------------------------------------------------------------------------------------------------
// STATIC TARGET
//-------------------------------------------------------------------------------------------------

type staticTarget struct {
	ips []string
}

// NewStaticTarget creates a new target which provides the given static IPs. IPs may be IPv4 or
// IPv6 addresses (and any combination thereof).
func NewStaticTarget(ips ...string) Target {
	return staticTarget{ips: ips}
}

func (t staticTarget) Targets(ctx context.Context, client client.Client, targets *string) ([]string, error) {
	return t.ips, nil
}

func (t staticTarget) NamespacedName() *types.NamespacedName {
	return nil
}
