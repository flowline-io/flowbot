package homelab

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

type composeDocument struct {
	Services map[string]composeService `yaml:"services"`
	Networks map[string]any            `yaml:"networks"`
}

type composeService struct {
	Image         string `yaml:"image"`
	ContainerName string `yaml:"container_name"`
	Ports         []any  `yaml:"ports"`
	Labels        any    `yaml:"labels"`
}

func ParseCompose(data []byte) ([]ComposeService, []string, []PortMapping, map[string]string, error) {
	var doc composeDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse compose: %w", err)
	}
	serviceNames := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	services := make([]ComposeService, 0, len(doc.Services))
	ports := make([]PortMapping, 0, len(doc.Services)*2)
	labels := make(map[string]string)
	for _, name := range serviceNames {
		svc := doc.Services[name]
		servicePorts := parsePorts(svc.Ports)
		services = append(services, ComposeService{
			Name:      name,
			Image:     svc.Image,
			Container: svc.ContainerName,
			Ports:     servicePorts,
		})
		ports = append(ports, servicePorts...)
		maps.Copy(labels, normalizeLabels(svc.Labels))
	}
	networks := make([]string, 0, len(doc.Networks))
	for name := range doc.Networks {
		networks = append(networks, name)
	}
	return services, networks, ports, labels, nil
}

func parsePorts(values []any) []PortMapping {
	result := make([]PortMapping, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			result = append(result, parsePortString(v))
		case map[string]any:
			result = append(result, parsePortMap(v))
		}
	}
	return result
}

func parsePortMap(value map[string]any) PortMapping {
	return PortMapping{
		Host:      stringMapValue(value, "host_ip"),
		HostPort:  stringMapValue(value, "published"),
		Container: stringMapValue(value, "target"),
		Protocol:  defaultProtocol(stringMapValue(value, "protocol")),
	}
}

func parsePortString(value string) PortMapping {
	protocol := "tcp"
	if before, after, ok := strings.Cut(value, "/"); ok {
		value = before
		protocol = after
	}
	parts := strings.Split(value, ":")
	switch len(parts) {
	case 1:
		return PortMapping{Container: parts[0], Protocol: protocol}
	case 2:
		return PortMapping{HostPort: parts[0], Container: parts[1], Protocol: protocol}
	default:
		return PortMapping{Host: strings.Join(parts[:len(parts)-2], ":"), HostPort: parts[len(parts)-2], Container: parts[len(parts)-1], Protocol: protocol}
	}
}

func stringMapValue(value map[string]any, key string) string {
	item, ok := value[key]
	if !ok || item == nil {
		return ""
	}
	switch v := item.(type) {
	case int:
		return strconv.Itoa(v)
	case uint64:
		return strconv.FormatUint(v, 10)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", item)
	}
}

// normalizeLabels converts Docker Compose labels from either map or list format
// into a map[string]string. List format entries use the "key=value" convention.
func normalizeLabels(raw any) map[string]string {
	result := make(map[string]string)
	switch v := raw.(type) {
	case map[string]any:
		for key, value := range v {
			result[key] = fmt.Sprintf("%v", value)
		}
	case []any:
		for _, item := range v {
			s := fmt.Sprintf("%v", item)
			key, val, found := strings.Cut(s, "=")
			if !found {
				result[s] = ""
				continue
			}
			result[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	}
	return result
}

func defaultProtocol(protocol string) string {
	if protocol == "" {
		return "tcp"
	}
	return protocol
}
