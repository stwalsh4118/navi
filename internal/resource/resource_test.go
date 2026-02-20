package resource

import (
	"os"
	"testing"
)

func TestReadRSSBytes_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	rss := readRSSBytes(pid)
	if rss <= 0 {
		t.Errorf("readRSSBytes(%d) = %d, want > 0 for current process", pid, rss)
	}
}

func TestReadRSSBytes_NonexistentPID(t *testing.T) {
	// PID 0 is the kernel scheduler, not readable; use a very high PID
	rss := readRSSBytes(999999999)
	if rss != 0 {
		t.Errorf("readRSSBytes(999999999) = %d, want 0", rss)
	}
}

func TestGetChildPIDs_PID1(t *testing.T) {
	// PID 1 (init/systemd) should have children on a running system
	children := getChildPIDs(1)
	if len(children) == 0 {
		t.Skip("PID 1 has no children (may be in a container)")
	}
	// Just verify we got valid PIDs
	for _, pid := range children {
		if pid <= 0 {
			t.Errorf("getChildPIDs(1) returned invalid PID: %d", pid)
		}
	}
}

func TestGetChildPIDs_NonexistentPID(t *testing.T) {
	children := getChildPIDs(999999999)
	if len(children) != 0 {
		t.Errorf("getChildPIDs(999999999) = %v, want empty", children)
	}
}

func TestProcessTreeRSS_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	rss := processTreeRSS(pid)
	if rss <= 0 {
		t.Errorf("processTreeRSS(%d) = %d, want > 0 for current process", pid, rss)
	}
}

func TestPageSize(t *testing.T) {
	if pageSize <= 0 {
		t.Errorf("pageSize = %d, want > 0", pageSize)
	}
	// Standard page sizes are 4096 or 65536
	if pageSize != 4096 && pageSize != 65536 && pageSize != 16384 {
		t.Logf("unexpected page size: %d (may be valid for this architecture)", pageSize)
	}
}
