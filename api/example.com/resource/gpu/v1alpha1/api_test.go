/*
 * Copyright 2025 The Kubernetes Authors.
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

package v1alpha1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCpuConfigNormalize(t *testing.T) {
	tests := map[string]struct {
		cpuConfig   *CpuConfig
		expected    *CpuConfig
		expectedErr error
	}{
		"nil CpuConfig": {
			cpuConfig:   nil,
			expectedErr: errors.New("config is 'nil'"),
		},
		"empty CpuConfig": {
			cpuConfig: &CpuConfig{},
			expected: &CpuConfig{
				Sharing: &CpuSharing{
					Strategy: RealTimeStrategy,
					RealTimeConfig: &RealTimeConfig{
						RuntimeUs: DefaultRuntimeUs,
						PeriodUs:  DefaultPeriodUs,
					},
				},
			},
		},
		"empty CpuConfig with Cfs": {
			cpuConfig: &CpuConfig{
				Sharing: &CpuSharing{
					Strategy: CfsStrategy,
				},
			},
			expected: &CpuConfig{
				Sharing: &CpuSharing{
					Strategy: CfsStrategy,
					CfsConfig: &CfsConfig{
						Shares: DefaultShares,
					},
				},
			},
		},
		"full CpuConfig": {
			cpuConfig: &CpuConfig{
				Sharing: &CpuSharing{
					Strategy: RealTimeStrategy,
					RealTimeConfig: &RealTimeConfig{
						RuntimeUs: 500000,  // 500ms
						PeriodUs:  1000000, // 1000ms
					},
					CfsConfig: &CfsConfig{
						Shares: 2048,
					},
				},
			},
			expected: &CpuConfig{
				Sharing: &CpuSharing{
					Strategy: RealTimeStrategy,
					RealTimeConfig: &RealTimeConfig{
						RuntimeUs: 500000,  // 500ms
						PeriodUs:  1000000, // 1000ms
					},
					CfsConfig: &CfsConfig{
						Shares: 2048,
					},
				},
			},
		},
		"default CpuConfig is already normalized": {
			cpuConfig: DefaultCpuConfig(),
			expected:  DefaultCpuConfig(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.cpuConfig.Normalize()
			assert.Equal(t, test.expected, test.cpuConfig)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
