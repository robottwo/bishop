//go:build darwin

package system

/*
#include <mach/mach.h>
#include <mach/mach_host.h>
#include <sys/sysctl.h>

// Get CPU load info using Mach API
int get_cpu_load_info(uint32_t *user, uint32_t *system, uint32_t *idle, uint32_t *nice) {
    host_cpu_load_info_data_t cpuinfo;
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
    kern_return_t status = host_statistics(mach_host_self(), HOST_CPU_LOAD_INFO,
                                           (host_info_t)&cpuinfo, &count);
    if (status != KERN_SUCCESS) {
        return -1;
    }
    *user = cpuinfo.cpu_ticks[CPU_STATE_USER];
    *system = cpuinfo.cpu_ticks[CPU_STATE_SYSTEM];
    *idle = cpuinfo.cpu_ticks[CPU_STATE_IDLE];
    *nice = cpuinfo.cpu_ticks[CPU_STATE_NICE];
    return 0;
}

// Get total physical memory
uint64_t get_total_memory() {
    int mib[2] = {CTL_HW, HW_MEMSIZE};
    uint64_t mem = 0;
    size_t len = sizeof(mem);
    sysctl(mib, 2, &mem, &len, NULL, 0);
    return mem;
}

// Get VM statistics for memory usage
int get_vm_stats(uint64_t *active, uint64_t *wired, uint64_t *compressed, uint32_t *page_size) {
    vm_statistics64_data_t vmstat;
    mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
    kern_return_t status = host_statistics64(mach_host_self(), HOST_VM_INFO64,
                                             (host_info64_t)&vmstat, &count);
    if (status != KERN_SUCCESS) {
        return -1;
    }

    // Get page size
    host_basic_info_data_t hostinfo;
    count = HOST_BASIC_INFO_COUNT;
    status = host_info(mach_host_self(), HOST_BASIC_INFO, (host_info_t)&hostinfo, &count);
    if (status != KERN_SUCCESS) {
        return -1;
    }

    *page_size = hostinfo.max_mem > 0 ? vm_page_size : 4096;
    *active = vmstat.active_count;
    *wired = vmstat.wire_count;
    *compressed = vmstat.compressor_page_count;
    return 0;
}
*/
import "C"

import (
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

	// RAM using Mach API (no external process)
	res.RAMTotal = uint64(C.get_total_memory())

	var active, wired, compressed C.uint64_t
	var pageSize C.uint32_t
	if C.get_vm_stats(&active, &wired, &compressed, &pageSize) == 0 {
		ps := uint64(pageSize)
		if ps == 0 {
			ps = 4096
		}
		// Memory Used ≈ Active + Wired + Compressed (similar to Activity Monitor)
		res.RAMUsed = (uint64(active) + uint64(wired) + uint64(compressed)) * ps
	}

	// CPU using Mach API (no external process)
	var user, system, idle, nice C.uint32_t
	if C.get_cpu_load_info(&user, &system, &idle, &nice) == 0 {
		total := uint64(user) + uint64(system) + uint64(idle) + uint64(nice)

		cpuMutex.Lock()
		if !lastCPUSampleTime.IsZero() && total > lastTotalTicks {
			deltaTotal := total - lastTotalTicks
			deltaIdle := uint64(idle) - lastIdleTicks
			if deltaTotal > 0 {
				res.CPUPercent = 100.0 * float64(deltaTotal-deltaIdle) / float64(deltaTotal)
			}
		}
		lastTotalTicks = total
		lastIdleTicks = uint64(idle)
		lastCPUSampleTime = time.Now()
		cpuMutex.Unlock()
	}

	return res
}
