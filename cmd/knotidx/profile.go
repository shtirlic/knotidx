package main

import (
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
)

func cpuprofile() func() {
	if !programProfiler {
		return func() {
		}
	}
	f, err := os.Create("cpuprofile.prof")
	if err != nil {
		slog.Error("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		slog.Error("could not start CPU profile: ", err)
	}
	return func() {
		pprof.StopCPUProfile()
		defer f.Close()
	}
}

func memprofile() {
	if !programProfiler {
		return
	}
	f, err := os.Create("memprofile.prof")
	if err != nil {
		slog.Error("could not create memory profile: ", err)
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		slog.Error("could not write memory profile: ", err)
	}
}
