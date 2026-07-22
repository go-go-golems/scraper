//go:build linux

package workflowv3runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type isolationCgroup struct{ path string }

func createIsolationCgroup(policy workflowv3.IsolationPolicy) (*isolationCgroup, error) {
	body, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return nil, err
	}
	var relative string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "0::") {
			relative = strings.TrimPrefix(line, "0::")
			break
		}
	}
	if relative == "" || !filepath.IsAbs(relative) {
		return nil, fmt.Errorf("unified cgroup path is unavailable")
	}
	current := filepath.Join("/sys/fs/cgroup", filepath.Clean(relative))
	parent, err := delegatedCgroupParent(current)
	if err != nil {
		return nil, err
	}
	path, err := os.MkdirTemp(parent, "workflowv3-")
	if err != nil {
		return nil, err
	}
	group := &isolationCgroup{path: path}
	cleanup := true
	defer func() {
		if cleanup {
			_ = group.Kill()
			_ = os.Remove(path)
		}
	}()
	for name, value := range map[string]string{
		"memory.max": strconv.FormatInt(policy.MemoryBytes, 10),
		"pids.max":   strconv.FormatInt(policy.MaxProcesses, 10),
	} {
		if err := os.WriteFile(filepath.Join(path, name), []byte(value), 0o600); err != nil {
			return nil, fmt.Errorf("configure cgroup %s: %w", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(path, "memory.swap.max")); err == nil {
		if err := os.WriteFile(filepath.Join(path, "memory.swap.max"), []byte("0"), 0o600); err != nil {
			return nil, fmt.Errorf("disable cgroup swap: %w", err)
		}
	}
	cleanup = false
	return group, nil
}

func delegatedCgroupParent(current string) (string, error) {
	root := filepath.Clean("/sys/fs/cgroup")
	for candidate := filepath.Clean(current); strings.HasPrefix(candidate, root); candidate = filepath.Dir(candidate) {
		body, err := os.ReadFile(filepath.Join(candidate, "cgroup.subtree_control"))
		if err == nil && strings.Contains(string(body), "memory") && strings.Contains(string(body), "pids") {
			probe, probeErr := os.MkdirTemp(candidate, ".workflowv3-probe-")
			if probeErr == nil {
				_ = os.Remove(probe)
				return candidate, nil
			}
		}
		if candidate == root {
			break
		}
	}
	return "", fmt.Errorf("no writable delegated cgroup with memory and pids controllers")
}

func (g *isolationCgroup) CPUUsageMicros() (int64, error) {
	body, err := os.ReadFile(filepath.Join(g.path, "cpu.stat"))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("cgroup cpu.stat has no usage_usec counter")
}

func (g *isolationCgroup) OOMKills() (int64, error) {
	body, err := os.ReadFile(filepath.Join(g.path, "memory.events"))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "oom_kill" {
			return strconv.ParseInt(fields[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("cgroup memory.events has no oom_kill counter")
}

func (g *isolationCgroup) Kill() error {
	if g == nil || g.path == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(g.path, "cgroup.kill")); err == nil {
		if err := os.WriteFile(filepath.Join(g.path, "cgroup.kill"), []byte("1"), 0o600); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (g *isolationCgroup) Close() error {
	if g == nil || g.path == "" {
		return nil
	}
	_ = g.Kill()
	var err error
	for attempt := 0; attempt < 50; attempt++ {
		err = os.Remove(g.path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		time.Sleep(2 * time.Millisecond)
	}
	return err
}
