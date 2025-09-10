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

// Validate ensures that CpuSharingStrategy has a valid set of values.
func (s CpuSharingStrategy) Validate() error {
	switch s {
	case RealTimeStrategy, CfsStrategy:
		return nil
	}
	return fmt.Errorf("unknown CPU sharing strategy: %v", s)
}

// Validate ensures that RealTimeConfig has a valid set of values.
func (c *RealTimeConfig) Validate() error {
	if c.RuntimeUs <= 0 {
		return fmt.Errorf("invalid runtime: %v, must be positive", c.RuntimeUs)
	}
	if c.PeriodUs <= 0 {
		return fmt.Errorf("invalid period: %v, must be positive", c.PeriodUs)
	}
	if c.RuntimeUs > c.PeriodUs {
		return fmt.Errorf("runtime (%v) cannot be greater than period (%v)", c.RuntimeUs, c.PeriodUs)
	}
	return nil
}

// Validate ensures that CfsConfig has a valid set of values.
func (c *CfsConfig) Validate() error {
	if c.Shares <= 0 {
		return fmt.Errorf("invalid shares: %v, must be positive", c.Shares)
	}
	return nil
}

// Validate ensures that CpuSharing has a valid set of values.
func (s *CpuSharing) Validate() error {
	if err := s.Strategy.Validate(); err != nil {
		return err
	}
	switch {
	case s.IsRealTime():
		return s.RealTimeConfig.Validate()
	case s.IsCfs():
		return s.CfsConfig.Validate()
	}
	return fmt.Errorf("invalid CPU sharing settings: %v", s)
}

// Validate ensures that CpuConfig has a valid set of values.
func (c *CpuConfig) Validate() error {
	if c.Sharing == nil {
		return fmt.Errorf("no sharing strategy set")
	}
	return c.Sharing.Validate()
}
