package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/shtirlic/knot/internal/config"
	"github.com/shtirlic/knot/internal/indexer"
	"github.com/shtirlic/knot/internal/store"
)

var (
	programErr      error
	programExitCode = 1                  // Exit code set to 1 by default
	programLevel    = new(slog.LevelVar) // Info by default

	gConf  *config.Config
	gStore store.Store

	// command      = flag.String("s", "", "startup command")
)

func addToIndex(s store.Store, path string) {
	slog.Info("Starting addToIndex", "path", path, "store", s)
	idx := indexer.NewIndexer(path)
	idx.NewIndex(s)
}

func main() {
	flag.Parse()

	// Set slog logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})))
	programLevel.Set(slog.LevelDebug)

	slog.Info("Starting knotd")

	// profile
	// debug.SetMemoryLimit(64 << 20)
	defer cpuprofile()()

	// Handle shutdown
	defer func() {
		slog.Info("Stiopping knotd")
		if programErr != nil && programExitCode != 0 {
			slog.Error("exit", "error", programErr)
		}
		memprofile()
		os.Exit(programExitCode)
	}()

	// Load default config and file config
	if gConf, programErr = reloadConfig(); programErr != nil {
		return
	}

	// watch for singals
	sgsCh := watchSignals()

	// start background ticker
	ticker := time.NewTicker(time.Second * time.Duration(gConf.Knotd.Interval))
	defer ticker.Stop()

	// Open the store
	if gStore, programErr = NewGlobalStore(); programErr != nil {
		return
	}
	defer gStore.Close()

	// Starting the main daemon loop
	// waiting for background events or os signals
loop:
	for {
		select {
		case <-ticker.C:
			slog.Debug("Got work?")
			addIndexers()
		case sig := <-sgsCh:
			// Handle recieved signals
			_, exit, err := handleSginal(sig)
			if err != nil {
				programExitCode = exit
				programErr = err
				break loop
			}
		}
	}
}

func addIndexers() {
	for _, idx := range gConf.Knotd.Indexer {
		for _, path := range idx.Paths {
			go addToIndex(gStore, path)
		}
	}
}

// Create and Open the store
func NewGlobalStore() (s store.Store, err error) {
	if s, err = store.NewStore(gConf.Knotd.Store); err != nil {
		slog.Error("Can't create/open the store", "store", gStore, "error", err)
		return
	}
	return
}

// (Re)Load default config and file config
func reloadConfig() (conf *config.Config, err error) {
	if conf, err = config.DefaultConfig().Load(); err != nil {
		slog.Error("Can't read config from toml files", "error", err)
		return
	}
	return
}

func getExitCode(sig os.Signal) int {
	return 128 + int(sig.(syscall.Signal))
}

// Handle the SIGHUP via reloading config and refresh the store
func handleHUP(sig os.Signal) (handled bool, exit int, err error) {
	slog.Info("Reloading...")
	handled = false
	exit = getExitCode(sig)
	if gConf, err = reloadConfig(); err != nil {
		return
	}
	if err = gStore.Close(); err != nil {
		return
	}
	if gStore, err = NewGlobalStore(); err != nil {
		return
	}
	handled = true
	exit = 0
	runtime.GC()
	slog.Info("Reload complete")
	return
}

func handleQuit(sig os.Signal) (bool, int, error) {
	// exit := getExitCode(sig)
	exit := 0
	return true, exit, fmt.Errorf("got signal: %v", sig)
}

// Handle the OS signals
func handleSginal(sig os.Signal) (bool, int, error) {
	slog.Info("Got signal", "signal", sig)
	exit := getExitCode(sig)
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
		return handleQuit(sig)
	case syscall.SIGHUP:
		return handleHUP(sig)
	default:
		return false, exit, fmt.Errorf("signal %v not handled", sig)
	}
}

// Notify of the OS signals
func watchSignals() (sgsCh chan os.Signal) {
	sgsCh = make(chan os.Signal, 1)
	signal.Notify(sgsCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	return
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
	}
}
