package cni

import (
	"fmt"
	"math"
	"strings"

	"github.com/MikeZappa87/kni-api/pkg/apis/runtime/beta"
	"github.com/containerd/cri-containerd/pkg/server/bandwidth"
	"github.com/containerd/go-cni"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
)

func cniNamespaceOpts(id, name, namespace, uid string, labels map[string]string, annotations map[string]string, extra map[string]string,
	 ports []*beta.PortMapping, dnsconfig *beta.DNSConfig) ([]cni.NamespaceOpts, error) {
	opts := []cni.NamespaceOpts{
		cni.WithLabels(toCNILabels(id, name, namespace, uid)),
		cni.WithCapability("io.kubernetes.cri.pod-annotations", annotations),
	}

	portMappings := toCNIPortMappings(ports)
	if len(portMappings) > 0 {
		opts = append(opts, cni.WithCapabilityPortMap(portMappings))
	}

	// Will return an error if the bandwidth limitation has the wrong unit
	// or an unreasonable value see validateBandwidthIsReasonable()
	bandWidth, err := toCNIBandWidth(annotations)
	if err != nil {
		return nil, err
	}
	if bandWidth != nil {
		opts = append(opts, cni.WithCapabilityBandWidth(*bandWidth))
	}

	dns := toCNIDNS(dnsconfig)
	if dns != nil {
		opts = append(opts, cni.WithCapabilityDNS(*dns))
	}

	if cgroup := extra["cgroupPath"]; cgroup != "" {
		opts = append(opts, cni.WithCapabilityCgroupPath(cgroup))
		log.Infof("cgroup: %s", cgroup)
	}

	return opts, nil
}

func toCNILabels(id, name, namespace, uid string) map[string]string {
	return map[string]string{
		"K8S_POD_NAMESPACE":          namespace,
		"K8S_POD_NAME":               name,
		"K8S_POD_INFRA_CONTAINER_ID": id,
		"K8S_POD_UID":                uid,
		"IgnoreUnknown":              "1",
	}
}

var minRsrc = resource.MustParse("1k")
var maxRsrc = resource.MustParse("1P")

func validateBandwidthIsReasonable(rsrc *resource.Quantity) error {
	if rsrc.Value() < minRsrc.Value() {
		return fmt.Errorf("resource is unreasonably small (< 1kbit)")
	}
	if rsrc.Value() > maxRsrc.Value() {
		return fmt.Errorf("resource is unreasonably large (> 1Pbit)")
	}
	return nil
}

// ExtractPodBandwidthResources extracts the ingress and egress from the given pod annotations
func ExtractPodBandwidthResources(podAnnotations map[string]string) (ingress, egress *resource.Quantity, err error) {
	if podAnnotations == nil {
		return nil, nil, nil
	}
	str, found := podAnnotations["kubernetes.io/ingress-bandwidth"]
	if found {
		ingressValue, err := resource.ParseQuantity(str)
		if err != nil {
			return nil, nil, err
		}
		ingress = &ingressValue
		if err := validateBandwidthIsReasonable(ingress); err != nil {
			return nil, nil, err
		}
	}
	str, found = podAnnotations["kubernetes.io/egress-bandwidth"]
	if found {
		egressValue, err := resource.ParseQuantity(str)
		if err != nil {
			return nil, nil, err
		}
		egress = &egressValue
		if err := validateBandwidthIsReasonable(egress); err != nil {
			return nil, nil, err
		}
	}
	return ingress, egress, nil
}


// toCNIBandWidth converts CRI annotations to CNI bandwidth.
func toCNIBandWidth(annotations map[string]string) (*cni.BandWidth, error) {
	ingress, egress, err := bandwidth.ExtractPodBandwidthResources(annotations)
	if err != nil {
		return nil, fmt.Errorf("reading pod bandwidth annotations: %w", err)
	}

	if ingress == nil && egress == nil {
		return nil, nil
	}

	bandWidth := &cni.BandWidth{}

	if ingress != nil {
		bandWidth.IngressRate = uint64(ingress.Value())
		bandWidth.IngressBurst = math.MaxUint32
	}

	if egress != nil {
		bandWidth.EgressRate = uint64(egress.Value())
		bandWidth.EgressBurst = math.MaxUint32
	}

	return bandWidth, nil
}

// toCNIPortMappings converts CRI port mappings to CNI.
func toCNIPortMappings(criPortMappings []*beta.PortMapping) []cni.PortMapping {
	var portMappings []cni.PortMapping
	for _, mapping := range criPortMappings {
		if mapping.HostPort <= 0 {
			continue
		}
		portMappings = append(portMappings, cni.PortMapping{
			HostPort:      mapping.HostPort,
			ContainerPort: mapping.ContainerPort,
			Protocol:      strings.ToLower(mapping.Protocol.String()),
			HostIP:        mapping.HostIp,
		})
	}
	return portMappings
}

// toCNIDNS converts CRI DNSConfig to CNI.
func toCNIDNS(dns *beta.DNSConfig) *cni.DNS {
	if dns == nil {
		return nil
	}
	return &cni.DNS{
		Servers:  dns.GetServers(),
		Searches: dns.GetSearches(),
		Options:  dns.GetOptions(),
	}
}
