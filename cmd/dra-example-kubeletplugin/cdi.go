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
	"strings"

	"sigs.k8s.io/dra-example-driver/pkg/consts"

	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdiparser "tags.cncf.io/container-device-interface/pkg/parser"
	cdispec "tags.cncf.io/container-device-interface/specs-go"
)

/*const (
	cdiVendor = "k8s." + consts.DriverName
	cdiClass  = "gpu"
	cdiKind   = cdiVendor + "/" + cdiClass

	cdiCommonDeviceName = "common"
)*/
const (
	cdiVendor = "k8s." + consts.DriverName
	cdiClass  = "cpu"
	cdiKind   = cdiVendor + "/" + cdiClass

	cdiCommonDeviceName = "common"
)

const hookBinaryPath = "/usr/local/bin/cpu-cgroup-hook"


type CDIHandler struct {
	cache *cdiapi.Cache
}

func NewCDIHandler(config *Config) (*CDIHandler, error) {
	cache, err := cdiapi.NewCache(
		cdiapi.WithSpecDirs(config.flags.cdiRoot),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new CDI cache: %w", err)
	}
	handler := &CDIHandler{
		cache: cache,
	}

	return handler, nil
}

func (cdi *CDIHandler) CreateCommonSpecFile() error {
	spec := &cdispec.Spec{
		Kind: cdiKind,
		Devices: []cdispec.Device{
			{
				Name: cdiCommonDeviceName,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						fmt.Sprintf("KUBERNETES_NODE_NAME=%s", os.Getenv("NODE_NAME")),
						fmt.Sprintf("DRA_RESOURCE_DRIVER_NAME=%s", consts.DriverName),
					},
				},
			},
		},
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	specName, err := cdiapi.GenerateNameForTransientSpec(spec, cdiCommonDeviceName)
	if err != nil {
		return fmt.Errorf("failed to generate Spec name: %w", err)
	}

	return cdi.cache.WriteSpec(spec, specName)
}

func (cdi *CDIHandler) CreateClaimSpecFile(claimUID string, devices PreparedDevices) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claimUID)

	spec := &cdispec.Spec{
		Kind:    cdiKind,
		Devices: []cdispec.Device{},
	}
	/*
    switch devices.Type() {
	case nascrd.RtCpuType:*/

	for _, device := range devices {
		claimEdits := cdiapi.ContainerEdits{
			ContainerEdits: &cdispec.ContainerEdits{
				Env: []string{
					fmt.Sprintf("CPU_DEVICE_%s_RESOURCE_CLAIM=%s", device.DeviceName[4:], claimUID),
				},
			},
		}
		claimEdits.Append(device.ContainerEdits)

		// Derive generic envs for the hook from per-device envs
		genericEnvs := deriveGenericCPUEnvs(device.DeviceName[4:], claimEdits.ContainerEdits.Env)
		allHookEnv := append([]string{}, claimEdits.ContainerEdits.Env...)
		allHookEnv = append(allHookEnv, genericEnvs...)

		// Add a prestart hook which runs on the host to set cgroup parameters based on env
		prestartHook := cdispec.Hook{
			Path: hookBinaryPath,
			Args: []string{"cpu-cgroup-hook"},
			Env:  allHookEnv,
		}

		cdiDevice := cdispec.Device{
			Name: fmt.Sprintf("%s-%s", claimUID, device.DeviceName),
			ContainerEdits: cdispec.ContainerEdits{
				Env: allHookEnv,
				Hooks: &cdispec.Hooks{
					Prestart: []cdispec.Hook{prestartHook},
				},
			},
		}

		spec.Devices = append(spec.Devices, cdiDevice)
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	return cdi.cache.WriteSpec(spec, specName)
}

func (cdi *CDIHandler) DeleteClaimSpecFile(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claimUID)
	return cdi.cache.RemoveSpec(specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string, devices []string) []string {
	cdiDevices := []string{
		cdiparser.QualifiedName(cdiVendor, cdiClass, cdiCommonDeviceName),
	}

	for _, device := range devices {
		cdiDevice := cdiparser.QualifiedName(cdiVendor, cdiClass, fmt.Sprintf("%s-%s", claimUID, device))
		cdiDevices = append(cdiDevices, cdiDevice)
	}

	return cdiDevices
}

// deriveGenericCPUEnvs takes per-device envs and produces generic env names for the hook to read
// e.g. CPU_DEVICE_0_RT_RUNTIME_US -> CPU_RT_RUNTIME_US
func deriveGenericCPUEnvs(deviceIndex string, envs []string) []string {
	prefix := fmt.Sprintf("CPU_DEVICE_%s_", deviceIndex)
	var out []string
	for _, e := range envs {
		if !strings.HasPrefix(e, prefix) {
			continue
		}
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		val := kv[1]
		trimmed := strings.TrimPrefix(key, prefix)
		// Only map the keys we expect the hook to consume
		switch trimmed {
		case "RT_RUNTIME_US":
			out = append(out, "CPU_RT_RUNTIME_US="+val)
		case "RT_PERIOD_US":
			out = append(out, "CPU_RT_PERIOD_US="+val)
		case "CFS_SHARES":
			out = append(out, "CPU_CFS_SHARES="+val)
		}
	}
	return out
}
