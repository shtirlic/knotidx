package idle

import (
	"runtime"
	"syscall"
)

type loadAvg struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

func Idle() float64 {
	return 100.0 * (1.0 - SysinfoAvg().Load5/float64(runtime.NumCPU()))
}

func SysinfoAvg() loadAvg {
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
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
