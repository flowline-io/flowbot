package docker

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
)

// parseGPUDeviceRequests parses a Docker --gpus style value into DeviceRequests.
// Adapted from github.com/docker/cli/opts GpuOpts to avoid depending on docker/cli.
func parseGPUDeviceRequests(value string) ([]container.DeviceRequest, error) {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	req := container.DeviceRequest{}
	seen := map[string]struct{}{}
	for _, field := range fields {
		if err := applyGPUField(&req, seen, field); err != nil {
			return nil, err
		}
	}

	if _, ok := seen["count"]; !ok && req.DeviceIDs == nil {
		req.Count = 1
	}
	if req.Options == nil {
		req.Options = make(map[string]string)
	}
	if req.Capabilities == nil {
		req.Capabilities = [][]string{{"gpu"}}
	}
	return []container.DeviceRequest{req}, nil
}

func applyGPUField(req *container.DeviceRequest, seen map[string]struct{}, field string) error {
	key, val, withValue := strings.Cut(field, "=")
	if _, ok := seen[key]; ok {
		return fmt.Errorf("gpu request key '%s' can be specified only once", key)
	}
	seen[key] = struct{}{}

	if !withValue {
		seen["count"] = struct{}{}
		count, err := parseGPUCount(key)
		if err != nil {
			return err
		}
		req.Count = count
		return nil
	}

	switch key {
	case "driver":
		req.Driver = val
		return nil
	case "count":
		count, err := parseGPUCount(val)
		if err != nil {
			return err
		}
		req.Count = count
		return nil
	case "device":
		req.DeviceIDs = strings.Split(val, ",")
		return nil
	case "capabilities":
		req.Capabilities = [][]string{append(strings.Split(val, ","), "gpu")}
		return nil
	case "options":
		r := csv.NewReader(strings.NewReader(val))
		optFields, err := r.Read()
		if err != nil {
			return fmt.Errorf("failed to read gpu options: %w", err)
		}
		req.Options = kvStringsToMap(optFields)
		return nil
	default:
		return fmt.Errorf("unexpected key '%s' in '%s'", key, field)
	}
}

func parseGPUCount(s string) (int, error) {
	if s == "all" {
		return -1, nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		var numErr *strconv.NumError
		if errors.As(err, &numErr) {
			err = numErr.Err
		}
		return 0, fmt.Errorf(`invalid count (%s): value must be either "all" or an integer: %w`, s, err)
	}
	return i, nil
}

func kvStringsToMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		k, v, _ := strings.Cut(value, "=")
		result[k] = v
	}
	return result
}
