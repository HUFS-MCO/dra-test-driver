/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func enumerateAllPossibleDevices(_ int) (AllocatableDevices, error) {
	cpuIDs, err := discoverOnlineCPUIds()
	if err != nil {
		return nil, err
	}

	alldevices := make(AllocatableDevices)
	for _, id := range cpuIDs {
		name := fmt.Sprintf("cpu-%d", id)
		device := resourceapi.Device{
			Name: name,
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
				"index": {
					IntValue: ptr.To(int64(id)),
				},
				"policy": {
					StringValue: ptr.To("RT|CFS"),
				},
				"driverVersion": {
					VersionValue: ptr.To("1.0.0"),
				},
			},
			Capacity: map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
				"cores": {
					Value: resource.MustParse("1"),
				},
			},
		}
		alldevices[device.Name] = device
	}
	return alldevices, nil
}

// discoverOnlineCPUIds returns the set of online CPU IDs on the node.
// It prefers parsing /sys/devices/system/cpu/online. If unavailable, it falls back to runtime.NumCPU.
func discoverOnlineCPUIds() ([]int, error) {
	const onlinePath = "/sys/devices/system/cpu/online"
	data, err := os.ReadFile(onlinePath)
	if err == nil {
		ids, perr := parseCPUList(strings.TrimSpace(string(data)))
		if perr == nil && len(ids) > 0 {
			return ids, nil
		}
	}
	// Fallback: use runtime.NumCPU.
	n := runtime.NumCPU()
	ids := make([]int, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, i)
	}
	return ids, nil
}

// parseCPUList parses a Linux CPU list string like "0-3,5,7-8" into a sorted list of IDs.
func parseCPUList(s string) ([]int, error) {
	if s == "" {
		return nil, fmt.Errorf("empty cpu list")
	}
	var ids []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "-") {
			rangeParts := strings.SplitN(p, "-", 2)
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid cpu range: %q", p)
			}
			for i := start; i <= end; i++ {
				ids = append(ids, i)
			}
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid cpu id: %q", p)
		}
		ids = append(ids, v)
	}
	return ids, nil
}
