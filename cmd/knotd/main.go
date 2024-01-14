package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/shtirlic/knot/internal/indexer"
	"github.com/shtirlic/knot/internal/store"
)

var (
	// command      = flag.String("s", "", "startup command")
	programLevel = new(slog.LevelVar) // Info by default

)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})))
	flag.Parse()
	programLevel.Set(slog.LevelDebug)

	// debug.SetMemoryLimit(64 << 20)
	defer cpuprofile()()
	defer memprofile()
	sgsCh := watchSignals()
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	slog.Info("Starting")
	go startIndexer()

loop:
	for {
		select {
		case <-ticker.C:
			slog.Debug("Got work?")
		case sig := <-sgsCh:
			slog.Warn("case sig", "signal", sig)
			break loop
		}
	}
}

func cpuprofile() func() {
	f, err := os.Create("cpuprofile.prof")
	if err != nil {
		slog.Error("could not create CPU profile: ", err)
		os.Exit(1)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		slog.Error("could not start CPU profile: ", err)
		os.Exit(1)

	}

	return func() {
		pprof.StopCPUProfile()
		defer f.Close() // error handling omitted for example
	}
}

func memprofile() {
	f, err := os.Create("memprofile.prof")
	if err != nil {
		slog.Error("could not create memory profile: ", err)
		os.Exit(1)
	}
	defer f.Close() // error handling omitted for example
	runtime.GC()    // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		slog.Error("could not write memory profile: ", err)
		os.Exit(1)

	}
}

func startIndexer() {
	idx := indexer.NewFsIndexer("/home/shtirlic", nil, nil)
	// idx.Run(store.NewInMemoryBadgerStore())
	idx.Run(store.NewDiskBadgerStore("store.knot"))
}

func watchSignals() (sgsCh chan os.Signal) {
	sgsCh = make(chan os.Signal, 1)
	signal.Notify(sgsCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return
}
