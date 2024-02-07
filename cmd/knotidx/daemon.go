package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"time"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/idle"
	"github.com/shtirlic/knotidx/internal/indexer"
	"github.com/shtirlic/knotidx/internal/store"
	"golang.org/x/sys/unix"
)

// Constants defining thresholds and intervals
const (
	idleThreshold   = 95.0          // Idle percentage threshold for triggering actions
	triggerInterval = 1 * time.Hour // Time interval for triggering actions
	// triggerInterval = 10 * time.Second // Alternative shorter time interval for testing
)

// Variables used for background tasks
var (
	ticker *time.Ticker // Ticker for periodic actions

	wg sync.WaitGroup // WaitGroup for coordinating goroutines

	quitCh   chan bool = make(chan bool) // Channel for quitting the main loop
	quitWtCh chan bool = make(chan bool) // Channel for quitting the watcher goroutine

	lastTriggerTime time.Time = time.UnixMicro(0) // Time of the last triggered action

	daemonConf  config.Config
	daemonStore store.Store
)

// stopTicker stops the background ticker
func stopTicker() {
	if ticker != nil {
		slog.Debug("Stopping background ticker")
		ticker.Stop()
	}
}

// waitJobs closes the watchers quit channel and waits for indexers jobs to finish.
func waitJobs() {
	slog.Debug("Closing the watchers quit channel")
	close(quitWtCh) // Close the channel to signal watcher goroutines to quit
	slog.Debug("Waiting for all indexers jobs to finish")
	wg.Wait() // Wait for all indexer jobs to finish
}

// daemonShutDown performs shutdown actions for the daemon.
func daemonShutDown() {
	stopTicker()     // Stop the background ticker
	waitJobs()       // Wait for all indexers jobs to finish
	stopGRPCServer() // Stop the gRPC server
}

// daemonStart initializes and starts the knotidx daemon.
func daemonStart(c config.Config, s store.Store) (int, error) {
	daemonConf = c
	daemonStore = s

	slog.Info("Starting knotidx daemon")

	// Set memory limit for debugging
	debug.SetMemoryLimit(384 << 20)
	defer cpuprofile()()

	// Set up channel for signal notifications
	sigCh := watchSignals()

	// Start background ticker
	ticker = newTicker(time.Duration(daemonConf.Interval))

	// Start gRPC server in a goroutine
	go startGRPCServer(daemonConf, daemonStore)

	var daemonErr error
	var daemonExitCode int

	// Main daemon loop
	for {
		select {
		case <-ticker.C:
			// Periodic work
			tick()
		case sig := <-sigCh:
			// Handle received signals
			_, exit, err := handleSginal(sig)
			if err != nil {
				daemonExitCode = exit
				daemonErr = err
				close(quitCh)
			}
		case <-quitCh:
			// Quit the main loop
			return daemonExitCode, daemonErr
		}
	}
}

// newTicker creates a new time.Ticker with the specified duration.
func newTicker(t time.Duration) *time.Ticker {
	return time.NewTicker(time.Second * t)
}

// tick is a function called during each tick of the background ticker.
// It logs information about the interval and triggers scheduled work.
func tick() {
	slog.Debug("Got work?", "interval", daemonConf.Interval)
	scheduleWork()
}

// scheduleWork is responsible for scheduling and triggering background work based on specified conditions.
func scheduleWork() {
	// Get the current system idle time
	idleTime := idle.Idle()

	slog.Debug("Load AVG", "load", idle.SysinfoAvg())
	slog.Debug("Idle time", "idle", idleTime)
	slog.Debug("Last work was at:", "date", lastTriggerTime)

	// Check if the conditions for triggering work are met
	if (idleTime >= idleThreshold && time.Since(lastTriggerTime) >= triggerInterval) || time.UnixMicro(0) == lastTriggerTime {

		// Wait for any unfinished jobs before scheduling new ones
		waitJobs()
		// Update the last trigger time to the current time
		lastTriggerTime = time.Now()
		slog.Info("Start addIndexers job", "time", lastTriggerTime, daemonStore)
		// Create a new quit channel for indexers
		quitWtCh = make(chan bool)
		// Start the addIndexers job
		addIndexers(daemonConf.Indexer, daemonStore, quitWtCh)
	}
}

// addWatcher is responsible for adding a watcher to the specified indexer and managing its lifecycle.
func addWatcher(idx indexer.Indexer, quitWtCh chan bool) {
	defer wg.Done()

	// Start the watcher for the indexer
	idx.Watch(quitWtCh)
}

// addToIndex is responsible for adding items to the index using the specified indexer.
func addToIndex(idx indexer.Indexer) {
	defer wg.Done()

	slog.Info("Starting updateIndex", "config", idx.Config())
	// Attempt to update the index using the specified indexer
	if err := idx.UpdateIndex(); err != nil {
		slog.Error("new index failed", "error", err, "indexer", idx)
	}
}

// addIndexers creates and starts indexers and watchers for each configuration in idxc.
// It launches goroutines to update the index and watch for changes in the background.
func addIndexers(idxc []config.IndexerConfig, s store.Store, qCh chan bool) {

	slog.Debug("Indexers", "idx count", len(idxc))

	// Iterate over each indexer configuration
	for _, idxConfig := range idxc {
		// Create indexers based on the configuration
		for _, idx := range indexer.NewIndexers(idxConfig, s) {

			// Launch a goroutine to update the index
			wg.Add(1)
			go addToIndex(idx)

			// Launch a goroutine to watch for changes
			wg.Add(1)
			go addWatcher(idx, qCh)
		}
	}
}

// newStore creates and opens a new store based on the provided configuration.
// It returns the created store and any error encountered during creation or opening.
func newStore(c config.StoreConfig) (store.Store, error) {
	// Attempt to create/open the store
	s, err := store.NewStore(c)
	if err != nil {
		slog.Error("Can't create/open the store", "store", s, "error", err)
		return nil, err
	}
	return s, nil
}

// resetScheduler resets the scheduler by updating the ticker interval and resetting the lastTriggerTime.
// It takes an integer parameter 'in' representing the new interval in seconds.
func resetScheduler(in int) {
	slog.Info("Scheduler reset", "interval", in)
	// Reset the ticker to the new interval
	ticker.Reset(time.Second * time.Duration(in))
	// Reset the last background run time
	lastTriggerTime = time.UnixMicro(0)
}

// getExitCode calculates the exit code based on the received OS signal.
// It adds 128 to the signal value to create a unique exit code for each signal.
// The resulting exit code is suitable for conveying the reason for program termination.
func getExitCode(sig os.Signal) int {
	return 128 + int(sig.(unix.Signal))
}

// handleHUP handles the SIGHUP signal, typically used for reloading configurations.
// It initiates a graceful shutdown of the daemon, closes the store, performs necessary cleanup,
// and then restarts the daemon with the updated configuration.
// If successful, it returns handled as true, exit as 0, and an error if any occurred.
func handleHUP(sig os.Signal) (handled bool, exit int, err error) {
	slog.Info("Reloading...")
	handled = false
	exit = getExitCode(sig)

	// Gracefully shut down the daemon
	daemonShutDown()

	// Close the store
	if err = daemonStore.Close(); err != nil {
		return
	}

	// Capture memory profile
	memprofile()

	// Restart the daemon with the updated configuration
	if daemonConf, daemonStore, err = startUp(); err != nil {
		return
	}

	// Create a new quitWtCh channel
	quitWtCh = make(chan bool)

	// Reset the scheduler with the new interval
	resetScheduler(daemonConf.Interval)

	// Start the gRPC server in a new goroutine
	go startGRPCServer(daemonConf, daemonStore)

	handled = true
	exit = 0
	slog.Info("Reload complete")
	return
}

// handleQuit handles termination signals (SIGINT, SIGTERM, SIGQUIT).
func handleQuit(sig os.Signal) (bool, int, error) {
	// exit := getExitCode(sig)
	exit := 0
	// Return a flag indicating the signal was handled, the exit code, and an error message
	return true, exit, fmt.Errorf("got signal: %v", sig)
}

// handleSignal handles OS signals by determining the appropriate action,
// and returning a flag indicating if the signal was handled, the exit code, and an error message if any.
func handleSginal(sig os.Signal) (bool, int, error) {
	slog.Info("Got signal", "signal", sig)
	// Get the exit code associated with the signal
	exit := getExitCode(sig)

	// Handle different signals
	switch sig {
	case unix.SIGINT, unix.SIGTERM, unix.SIGQUIT:
		return handleQuit(sig)
	case unix.SIGHUP:
		return handleHUP(sig)
	default:
		return false, exit, fmt.Errorf("signal %v not handled", sig)
	}
}

// watchSignals sets up a channel for notifying the program of OS signals.
// It creates a buffered channel, registers it to receive specified signals (SIGINT, SIGTERM, SIGQUIT, SIGHUP),
// and returns the channel for signal notifications.
func watchSignals() chan os.Signal {
	// Create a buffered channel for signal notifications
	sgsCh := make(chan os.Signal, 1)

	// Register the channel to receive specified signals (SIGINT, SIGTERM, SIGQUIT, SIGHUP)
	signal.Notify(sgsCh,
		unix.SIGINT,
		unix.SIGTERM,
		unix.SIGQUIT,
		unix.SIGHUP,
	)
	return sgsCh
}
