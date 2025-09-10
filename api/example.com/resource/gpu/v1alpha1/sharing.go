/*
 * Copyright 2024 The Kubernetes Authors.
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
	"fmt"
)

// These constants represent the different Sharing strategies.
const (
	RealTimeStrategy CpuSharingStrategy = "RealTime"
	CfsStrategy      CpuSharingStrategy = "Cfs"
)

// These constants represent the different CPU sharing configurations.
const (
	DefaultRuntimeUs int64 = 950000  // 950ms
	DefaultPeriodUs  int64 = 1000000 // 1000ms (1s)
	DefaultShares    int64 = 1024
)

// CpuSharingStrategy defines the valid Sharing strategies as a string.
type CpuSharingStrategy string

// CpuSharing holds the current sharing strategy for CPUs and its settings.
// If DeviceClass and ResourceClaim set this, then the strategy from the claim
// is used. If multiple configurations set this, then the last one is used.
type CpuSharing struct {
	Strategy        CpuSharingStrategy `json:"strategy"`
	RealTimeConfig  *RealTimeConfig    `json:"realTimeConfig,omitempty"`
	CfsConfig       *CfsConfig         `json:"cfsConfig,omitempty"`
}

// RealTimeConfig provides the settings for the RealTime strategy.
type RealTimeConfig struct {
	// RuntimeUs specifies the CPU runtime in microseconds for real-time scheduling
	RuntimeUs int64 `json:"runtimeUs,omitempty"`
	// PeriodUs specifies the CPU period in microseconds for real-time scheduling
	PeriodUs int64 `json:"periodUs,omitempty"`
}

// CfsConfig provides the settings for the CFS (Completely Fair Scheduler) strategy.
type CfsConfig struct {
	// Shares indicates the CPU shares for CFS scheduling
	Shares int64 `json:"shares,omitempty"`
}

// IsRealTime checks if the RealTime strategy is applied.
func (s *CpuSharing) IsRealTime() bool {
	if s == nil {
		return false
	}
	return s.Strategy == RealTimeStrategy
}

// IsCfs checks if the CFS strategy is applied.
func (s *CpuSharing) IsCfs() bool {
	if s == nil {
		return false
	}
	return s.Strategy == CfsStrategy
}

// GetRealTimeConfig returns the real-time config that applies to the given strategy.
func (s *CpuSharing) GetRealTimeConfig() (*RealTimeConfig, error) {
	if s == nil {
		return nil, fmt.Errorf("no sharing set to get config from")
	}
	if s.Strategy != RealTimeStrategy {
		return nil, fmt.Errorf("strategy is not set to '%v'", RealTimeStrategy)
	}
	if s.CfsConfig != nil {
		return nil, fmt.Errorf("cannot use CfsConfig with the '%v' strategy", RealTimeStrategy)
	}
	return s.RealTimeConfig, nil
}

// GetCfsConfig returns the CFS config that applies to the given strategy.
func (s *CpuSharing) GetCfsConfig() (*CfsConfig, error) {
	if s == nil {
		return nil, fmt.Errorf("no sharing set to get config from")
	}
	if s.Strategy != CfsStrategy {
		return nil, fmt.Errorf("strategy is not set to '%v'", CfsStrategy)
	}
	if s.RealTimeConfig != nil {
		return nil, fmt.Errorf("cannot use RealTimeConfig with the '%v' strategy", CfsStrategy)
	}
	return s.CfsConfig, nil
}
