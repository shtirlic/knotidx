package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/idle"
	"github.com/shtirlic/knotidx/internal/indexer"
	"github.com/shtirlic/knotidx/internal/store"
)

const (
	idleThreshold   = 95.0 // idle percentage
	triggerInterval = 1 * time.Hour
	// triggerInterval = 10 * time.Second
)

var (
	ticker *time.Ticker

	wg sync.WaitGroup

	quitCh   chan bool = make(chan bool)
	quitWtCh chan bool = make(chan bool)

	lastTriggerTime time.Time = time.UnixMicro(0)
)

// Stop the background ticker
func stopTicker() {
	if ticker != nil {
		slog.Debug("Stopping background ticker")
		ticker.Stop()
	}
}

// Close watchers channel and wait for indexers jobs
func waitJobs() {
	slog.Debug("Closing the watchers quit channel")
	close(quitWtCh)
	slog.Debug("Waiting for all indexers jobs to finish")
	wg.Wait()
}

func daemonShutDown() {
	stopGrpcServer()
	stopTicker()
	waitJobs()
}

func daemonStart() {

	slog.Info("Starting knotidx daemon")

	// profile
	debug.SetMemoryLimit(384 << 20)
	defer cpuprofile()()

	// Set channel notify for singals
	sigCh := watchSignals()

	// Start background ticker
	ticker = newTicker(time.Duration(gConf.Interval))

	// start grpc server
	go startGrpcServer(gConf.Grpc)

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
				close(quitCh)
			}
		case <-quitCh:
			return
		}
	}
}

func newTicker(t time.Duration) *time.Ticker {
	return time.NewTicker(time.Second * t)
}

func tick() {
	slog.Debug("Got work?", "interval", gConf.Interval)
	scheduleWork()
}

func scheduleWork() {
	idleTime := idle.Idle()

	slog.Debug("Load AVG", "load", idle.SysinfoAvg())
	slog.Debug("Idle time", "idle", idleTime)
	slog.Debug("Last work was at:", "date", lastTriggerTime)

	if (idleTime >= idleThreshold && time.Since(lastTriggerTime) >= triggerInterval) || time.UnixMicro(0) == lastTriggerTime {

		// wait for unfinished jobs before schedule a new ones
		waitJobs()
		lastTriggerTime = time.Now()
		slog.Info("Start addIndexers job", "time", lastTriggerTime)
		quitWtCh = make(chan bool)
		addIndexers(gConf.Indexer, gStore, quitWtCh)
	}

}

// Watcher goroutine
func addWatcher(idx indexer.Indexer, quitWtCh chan bool) {
	defer wg.Done()

	idx.Watch(quitWtCh)
}

// Indexer goroutine
func addToIndex(idx indexer.Indexer) {
	defer wg.Done()

	slog.Info("Starting updateIndex", "config", idx.Config())
	if err := idx.UpdateIndex(); err != nil {
		slog.Error("new index failed", "error", err, "indexer", idx)
	}

}

func addIndexers(idxConfigs []config.IndexerConfig, s store.Store, quitWtCh chan bool) {

	slog.Debug("Indexers", "idx count", len(idxConfigs))

	for _, idxConfig := range idxConfigs {
		for _, idx := range indexer.NewIndexers(idxConfig, s) {

			// Fire goroutine index
			wg.Add(1)
			go addToIndex(idx)

			// Fire goroutine watcher
			wg.Add(1)
			go addWatcher(idx, quitWtCh)
		}
	}
}

// Create and Open the store
func newStore(conf config.StoreConfig) (s store.Store, err error) {
	if s, err = store.NewStore(conf); err != nil {
		slog.Error("Can't create/open the store", "store", s, "error", err)
		return
	}
	return
}

func resetScheduler(interval int) {
	slog.Info("Scheduler reset", "interval", interval)
	// Reset ticker to the new config value
	ticker.Reset(time.Second * time.Duration(interval))
	// Reset last background run time
	lastTriggerTime = time.UnixMicro(0)
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

	daemonShutDown()

	if err = gStore.Close(); err != nil {
		return
	}

	memprofile()

	if gConf, gStore, err = startUp(); err != nil {
		return
	}

	quitWtCh = make(chan bool)

	resetScheduler(gConf.Interval)

	// start grpc server
	go startGrpcServer(gConf.Grpc)

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
