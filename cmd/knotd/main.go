package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/shtirlic/knot/internal/config"
	"github.com/shtirlic/knot/internal/idle"
	"github.com/shtirlic/knot/internal/indexer"
	"github.com/shtirlic/knot/internal/store"
)

const (
	idleThreshold   = 90.0 // idle percentage
	triggerInterval = 1 * time.Hour
)

var (
	version = "development"
	commit  string
	date    = time.Now().String()

	programErr      error
	programExitCode = 1                  // Exit code set to 1 by default
	programLevel    = new(slog.LevelVar) // Info by default

	ticker *time.Ticker
	gConf  config.Config
	gStore store.Store
	wg     sync.WaitGroup

	lastTriggerTime time.Time

	// command      = flag.String("s", "", "startup command")
)

func main() {
	flag.Parse()

	// Set slog logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})))
	programLevel.Set(slog.LevelDebug)

	slog.Info("Starting knotd")
	slog.Info("Build", "version", version, "commit", commit, "date", date)

	// profile
	// debug.SetMemoryLimit(64 << 20)
	defer cpuprofile()()

	// Handle shutdown
	defer shutdown()

	// Set channel notify for singals
	sigCh := watchSignals()

	// Load configs and open store
	if gConf, gStore, programErr = startUp(); programErr != nil {
		return
	}

	// Start background ticker
	ticker = newTicker(time.Duration(gConf.Knotd.Interval))

	// Quit channel
	quit := make(chan bool)

	// Start the main daemon loop
	// Wait for background events and OS signals
	for {
		select {
		case <-ticker.C:
			// Do periodic work
			tick()
		case sig := <-sigCh:
			// Handle recieved signals
			_, exit, err := handleSginal(sig)
			if err != nil {
				programExitCode = exit
				programErr = err
				close(quit)
			}
		case <-quit:
			return
		}
	}
}

func newTicker(t time.Duration) *time.Ticker {
	return time.NewTicker(time.Second * t)
}

func tick() {
	slog.Debug("Got work?")
	scheduleWork()
}

func scheduleWork() {
	idleTime := idle.Idle()

	slog.Debug("Load AVG", "load", idle.SysinfoAvg())
	slog.Info("Idle time", "idle", idleTime)
	slog.Debug("Last work was at:", "date", lastTriggerTime)

	if idleTime >= idleThreshold && time.Since(lastTriggerTime) >= triggerInterval {
		lastTriggerTime = time.Now()
		slog.Info("Doing work 1 time per hour")
		addIndexers(gConf.Knotd.Indexer, gStore)
	}

}

func addToIndex(idx indexer.Indexer) {
	defer wg.Done()

	slog.Info("Starting addToIndex", "type", idx.Type())
	if err := idx.NewIndex(); err != nil {
		slog.Error("new index failed", "error", err, "indexer", idx)
	}

}

func addIndexers(idxConfigs []config.IndexerConfig, s store.Store) {
	for _, idxConfig := range idxConfigs {
		for _, idx := range indexer.NewIndexers(idxConfig, s) {

			// Fire goroutines for the actual indexing
			wg.Add(1)
			go addToIndex(idx)
		}
	}
}

// (Re)Load default config and file config
func reloadConfig() (conf config.Config, err error) {
	if conf, err = config.DefaultConfig().Load(); err != nil {
		slog.Error("Can't read config from toml files", "error", err)
		return
	}
	return
}

// Create and Open the store
func newGlobalStore(conf config.StoreConfig) (s store.Store, err error) {
	if s, err = store.NewStore(conf); err != nil {
		slog.Error("Can't create/open the store", "store", s, "error", err)
		return
	}
	return
}

// Do daemon shutdown
func shutdown() {
	slog.Info("Stoopping knotd")

	ticker.Stop()
	wg.Wait()
	gStore.Close()

	if programErr != nil && programExitCode != 0 {
		slog.Error("exit", "error", programErr)
	}
	memprofile()
	os.Exit(programExitCode)
}

// Startup seq reloading configs and create/open store
func startUp() (conf config.Config, s store.Store, err error) {
	// Load default config and file config
	if conf, err = reloadConfig(); err != nil {
		return
	}
	// Open the store
	if s, err = newGlobalStore(conf.Knotd.Store); err != nil {
		return
	}
	return
}

// Calculate the exit code
func getExitCode(sig os.Signal) int {
	return 128 + int(sig.(syscall.Signal))
}

// Handle the SIGHUP via reloading config and refresh the store
func handleHUP(sig os.Signal) (handled bool, exit int, err error) {
	slog.Info("Reloading...")
	handled = false
	exit = getExitCode(sig)

	// stop the background ticker
	ticker.Stop()

	// wait for indexer tasks to finish
	wg.Wait()

	if err = gStore.Close(); err != nil {
		return
	}
	runtime.GC()
	// memprofile()

	if gConf, gStore, err = startUp(); err != nil {
		return
	}
	ticker.Reset(time.Second * time.Duration(gConf.Knotd.Interval))

	handled = true
	exit = 0
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
