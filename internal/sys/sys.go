package sys

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"time"
)

// ProjectContext holds deep architectural insights about a directory
type ProjectContext struct {
	RootPath    string            `json:"root_path"`
	GitSHA      string            `json:"git_sha"`
	LogicalMap  map[string]string `json:"logical_map"` // Key insights like "entrypoint": "main.go"
	LastIndexed time.Time         `json:"last_indexed"`
}

// Snapshot represents the current system state
type Snapshot struct {
	CPUUsage    float64
	MemoryUsage float64
	WorkingDir  string
}

// Monitor provides system awareness
type Monitor struct{}

func NewMonitor() *Monitor {
	return &Monitor{}
}

// GetSnapshot returns a current snapshot of system resources
func (m *Monitor) GetSnapshot() (Snapshot, error) {
	c, err := cpu.Percent(0, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("getting cpu percent: %w", err)
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return Snapshot{}, fmt.Errorf("getting virtual memory: %w", err)
	}

	wd, _ := os.Getwd()

	return Snapshot{
		CPUUsage:    c[0],
		MemoryUsage: vm.UsedPercent,
		WorkingDir:  wd,
	}, nil
}

