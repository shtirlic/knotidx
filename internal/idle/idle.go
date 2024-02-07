package idle

import (
	"runtime"

	"golang.org/x/sys/unix"
)

// loadAvg represents system load averages over different time intervals.
type loadAvg struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

// Idle calculates the idle percentage based on the system load average over
// the last 5 minutes and the number of CPUs.
// The formula used is: Idle = 100 * (1 - LoadAverage / NumCPUs)
func Idle() float64 {
	return 100.0 * (1.0 - SysinfoAvg().Load5/float64(runtime.NumCPU()))
}

// SysinfoAvg retrieves system load averages (1, 5, and 15 minutes) and returns
// them as a loadAvg struct.
func SysinfoAvg() loadAvg {
	var info unix.Sysinfo_t
	err := unix.Sysinfo(&info)
	if err != nil {
		return loadAvg{}
	}

	const si_load_shift = 16
	return loadAvg{
		Load1:  float64(info.Loads[0]) / float64(1<<si_load_shift),
		Load5:  float64(info.Loads[1]) / float64(1<<si_load_shift),
		Load15: float64(info.Loads[2]) / float64(1<<si_load_shift),
	}
}
