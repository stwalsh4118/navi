package resource

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// pageSize is the system memory page size in bytes, cached at init.
var pageSize = int64(os.Getpagesize())

// SessionRSS returns the total RSS in bytes for all processes in a tmux session's
// process tree. It gets root pane PIDs via tmux, then recursively walks /proc to
// sum RSS for every descendant. Returns 0 if the session has no panes or on any error.
func SessionRSS(sessionName string) int64 {
	pids := getPanePIDs(sessionName)
	if len(pids) == 0 {
		return 0
	}

	var total int64
	for _, pid := range pids {
		total += processTreeRSS(pid)
	}
	return total
}

// getPanePIDs returns the root shell PIDs for all panes in a tmux session.
func getPanePIDs(sessionName string) []int {
	cmd := exec.Command("tmux", "list-panes", "-s", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var pids []int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

// processTreeRSS recursively sums RSS for a process and all its descendants.
func processTreeRSS(pid int) int64 {
	rss := readRSSBytes(pid)
	for _, child := range getChildPIDs(pid) {
		rss += processTreeRSS(child)
	}
	return rss
}

// readRSSBytes reads a single process's RSS from /proc/<pid>/statm.
// The second field of statm is RSS in pages. Returns 0 on any error.
func readRSSBytes(pid int) int64 {
	path := filepath.Join("/proc", strconv.Itoa(pid), "statm")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}

	rssPages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	return rssPages * pageSize
}

// getChildPIDs returns the child PIDs for a process by reading /proc/<pid>/task/*/children.
// Returns an empty slice on any error or if no children exist.
func getChildPIDs(pid int) []int {
	pidStr := strconv.Itoa(pid)
	pattern := filepath.Join("/proc", pidStr, "task", "*", "children")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}

	seen := make(map[int]bool)
	var children []int

	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		for _, field := range strings.Fields(string(data)) {
			childPID, err := strconv.Atoi(field)
			if err != nil {
				continue
			}
			if !seen[childPID] {
				seen[childPID] = true
				children = append(children, childPID)
			}
		}
	}

	return children
}
