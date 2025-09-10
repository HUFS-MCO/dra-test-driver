package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ociState struct {
	Pid int `json:"pid"`
}

func main() {
	if err := run(os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "cpu-cgroup-hook error: %v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader) error {
	var state ociState
	if err := json.NewDecoder(bufio.NewReader(r)).Decode(&state); err != nil {
		return fmt.Errorf("decode OCI state: %w", err)
	}
	if state.Pid <= 0 {
		return fmt.Errorf("invalid pid in OCI state: %d", state.Pid)
	}

	cgPaths, isV2, err := getCgroupPaths(state.Pid)
	if err != nil {
		return fmt.Errorf("get cgroup paths: %w", err)
	}

	// Read desired settings from env
	rtRuntime := os.Getenv("CPU_RT_RUNTIME_US")
	rtPeriod := os.Getenv("CPU_RT_PERIOD_US")
	cfsShares := os.Getenv("CPU_CFS_SHARES")

	if isV2 {
		base := cgPaths[""]
		// Prefer cpu.rt_* if present (custom kernel), else fall back to cpu.max
		if rtRuntime != "" || rtPeriod != "" {
			if rtRuntime == "" || rtPeriod == "" {
				return errors.New("both CPU_RT_RUNTIME_US and CPU_RT_PERIOD_US must be set for RT")
			}
			rtRuntimePath := filepath.Join(base, "cpu.rt_runtime_us")
			rtPeriodPath := filepath.Join(base, "cpu.rt_period_us")
			if fileExists(rtRuntimePath) && fileExists(rtPeriodPath) {
				if err := os.WriteFile(rtRuntimePath, []byte(rtRuntime), 0644); err != nil {
					return fmt.Errorf("write cpu.rt_runtime_us: %w", err)
				}
				if err := os.WriteFile(rtPeriodPath, []byte(rtPeriod), 0644); err != nil {
					return fmt.Errorf("write cpu.rt_period_us: %w", err)
				}
			} else {
				cpuMax := filepath.Join(base, "cpu.max")
				if err := os.WriteFile(cpuMax, []byte(fmt.Sprintf("%s %s", rtRuntime, rtPeriod)), 0644); err != nil {
					return fmt.Errorf("write cpu.max: %w", err)
				}
			}
		}
		if cfsShares != "" {
			w, err := sharesToWeight(cfsShares)
			if err != nil {
				return err
			}
			cpuWeight := filepath.Join(base, "cpu.weight")
			if err := os.WriteFile(cpuWeight, []byte(strconv.Itoa(w)), 0644); err != nil {
				return fmt.Errorf("write cpu.weight: %w", err)
			}
		}
		return nil
	}

	// cgroup v1
	cpuPath, ok := cgPaths["cpu"]
	if !ok {
		return fmt.Errorf("cpu controller path not found in cgroup v1")
	}
	if rtRuntime != "" {
		if err := os.WriteFile(filepath.Join(cpuPath, "cpu.rt_runtime_us"), []byte(rtRuntime), 0644); err != nil {
			return fmt.Errorf("write cpu.rt_runtime_us: %w", err)
		}
	}
	if rtPeriod != "" {
		if err := os.WriteFile(filepath.Join(cpuPath, "cpu.rt_period_us"), []byte(rtPeriod), 0644); err != nil {
			return fmt.Errorf("write cpu.rt_period_us: %w", err)
		}
	}
	if cfsShares != "" {
		if err := os.WriteFile(filepath.Join(cpuPath, "cpu.shares"), []byte(cfsShares), 0644); err != nil {
			return fmt.Errorf("write cpu.shares: %w", err)
		}
	}
	return nil
}

// getCgroupPaths returns controller->path for v1 or base path for v2 (key "").
func getCgroupPaths(pid int) (map[string]string, bool, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return nil, false, fmt.Errorf("read /proc/<pid>/cgroup: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	paths := map[string]string{}
	isV2 := false
	for _, l := range lines {
		parts := strings.SplitN(l, ":", 3)
		if len(parts) != 3 {
			continue
		}
		controllers := parts[1]
		p := parts[2]
		if controllers == "" { // cgroup v2 unified
			isV2 = true
			paths[""] = filepath.Join("/sys/fs/cgroup", p)
			continue
		}
		for _, c := range strings.Split(controllers, ",") {
			paths[c] = filepath.Join("/sys/fs/cgroup", c, p)
		}
	}
	return paths, isV2, nil
}

// sharesToWeight converts cgroup v1 cpu.shares to v2 cpu.weight (1..10000)
func sharesToWeight(sharesStr string) (int, error) {
	shares, err := strconv.Atoi(sharesStr)
	if err != nil || shares <= 0 {
		return 0, fmt.Errorf("invalid shares: %q", sharesStr)
	}
	// Rough proportional mapping around default shares=1024 -> weight~100
	w := int(float64(shares) * (100.0 / 1024.0))
	if w < 1 {
		w = 1
	}
	if w > 10000 {
		w = 10000
	}
	return w, nil
}

func fileExists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}
