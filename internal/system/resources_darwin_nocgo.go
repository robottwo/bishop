//go:build darwin && !cgo

package system

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	lastCPUSampleTime time.Time
	lastTotalTicks    uint64
	lastIdleTicks     uint64
	cpuMutex          sync.Mutex
)

func getResources() *Resources {
	res := &Resources{
		Timestamp: time.Now(),
	}

	// Get total memory using sysctl
	// sysctl -n hw.memsize returns total physical memory in bytes
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil {
			res.RAMTotal = val
		}
	}

	// Get memory usage using vm_stat
	// vm_stat outputs page-based memory statistics
	if out, err := exec.Command("vm_stat").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		var pageSize uint64 = 4096 // Default page size
		var pagesActive, pagesWired, pagesCompressed uint64

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Parse page size from first line: "Mach Virtual Memory Statistics: (page size of 4096 bytes)"
			if strings.Contains(line, "page size of") {
				if start := strings.Index(line, "page size of "); start != -1 {
					sub := line[start+13:]
					if end := strings.Index(sub, " "); end != -1 {
						if val, err := strconv.ParseUint(sub[:end], 10, 64); err == nil {
							pageSize = val
						}
					}
				}
				continue
			}

			// Parse statistics lines like "Pages active:                  123456."
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			valStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
			val, err := strconv.ParseUint(valStr, 10, 64)
			if err != nil {
				continue
			}

			switch key {
			case "Pages active":
				pagesActive = val
			case "Pages wired down":
				pagesWired = val
			case "Pages occupied by compressor":
				pagesCompressed = val
			}
		}

		// Memory Used ≈ Active + Wired + Compressed (similar to Activity Monitor)
		res.RAMUsed = (pagesActive + pagesWired + pagesCompressed) * pageSize
	}

	// Get CPU usage using top in non-interactive mode
	// top -l 1 -n 0 outputs one sample with no processes
	if out, err := exec.Command("top", "-l", "1", "-n", "0").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			// Look for line like: "CPU usage: 5.26% user, 10.52% sys, 84.21% idle"
			if strings.HasPrefix(line, "CPU usage:") {
				// Extract idle percentage
				if idleIdx := strings.Index(line, "% idle"); idleIdx != -1 {
					// Find the start of the idle value
					sub := line[:idleIdx]
					if lastComma := strings.LastIndex(sub, ", "); lastComma != -1 {
						idleStr := strings.TrimSpace(sub[lastComma+2:])
						if idleVal, err := strconv.ParseFloat(idleStr, 64); err == nil {
							res.CPUPercent = 100.0 - idleVal
						}
					} else if lastSpace := strings.LastIndex(sub, " "); lastSpace != -1 {
						idleStr := strings.TrimSpace(sub[lastSpace+1:])
						if idleVal, err := strconv.ParseFloat(idleStr, 64); err == nil {
							res.CPUPercent = 100.0 - idleVal
						}
					}
				}
				break
			}
		}
	}

	return res
}
