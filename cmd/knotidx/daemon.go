package main

import (
	"context"
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

type Daemon struct {
	ticker          *time.Ticker // Ticker for periodic actions
	cancelJobs      context.CancelFunc
	cancelContext   context.Context
	wg              sync.WaitGroup // WaitGroup for coordinating goroutines
	lastTriggerTime time.Time      // Time of the last triggered action
	config          config.Config
	store           store.Store
	grpcServer      *GRPServer
}

func NewDaemon(c config.Config, s store.Store) *Daemon {
	return &Daemon{
		config:          c,
		store:           s,
		lastTriggerTime: time.UnixMicro(0),
		grpcServer:      NewGRPCServer(c, s),
	}
}

// stopTicker stops the background ticker
func (d *Daemon) stopTicker() {
	if d.ticker != nil {
		slog.Debug("Stopping background ticker")
		d.ticker.Stop()
	}
}

// waitJobs closes the watchers quit channel and waits for indexers jobs to finish.
func (d *Daemon) waitJobs() {
	slog.Debug("Waiting for all indexers jobs to finish")
	d.wg.Wait() // Wait for all indexer jobs to finish
}

// daemonShutDown performs shutdown actions for the daemon.
func (d *Daemon) ShutDown() {
	d.stopTicker() // Stop the background ticker
	d.cancelJobs()
	d.waitJobs()        // Wait for all indexers jobs to finish
	d.grpcServer.Stop() // Stop the gRPC server
}

// daemonStart initializes and starts the knotidx daemon.
func (d *Daemon) Start() (int, error) {

	slog.Info("Starting knotidx daemon")

	// Set memory limit for debugging
	debug.SetMemoryLimit(384 << 20)
	defer cpuprofile()()

	d.cancelContext, d.cancelJobs = context.WithCancel(context.Background())

	// Set up channel for signal notifications
	sigCh := d.watchSignals()

	// Start background ticker
	d.ticker = d.newTicker(time.Duration(d.config.Interval))

	// Start gRPC server in a goroutine
	go d.grpcServer.Start()

	var daemonErr error
	var daemonExitCode int

	// Main daemon loop
	for {
		select {
		case <-d.ticker.C:
			// Periodic work
			d.tick()
		case sig := <-sigCh:
			// Handle received signals
			_, exit, err := d.handleSginal(sig)
			if err != nil {
				daemonExitCode = exit
				daemonErr = err
				d.cancelJobs()
			}
		case <-d.cancelContext.Done():
			// Quit the main loop
			return daemonExitCode, daemonErr
		}
	}
}

// newTicker creates a new time.Ticker with the specified duration.
func (d *Daemon) newTicker(t time.Duration) *time.Ticker {
	return time.NewTicker(time.Second * t)
}

// tick is a function called during each tick of the background ticker.
// It logs information about the interval and triggers scheduled work.
func (d *Daemon) tick() {
	slog.Debug("Got work?", "interval", d.config.Interval)
	d.scheduleWork()
}

// scheduleWork is responsible for scheduling and triggering background work based on specified conditions.
func (d *Daemon) scheduleWork() {
	// Get the current system idle time
	idleTime := idle.Idle()

	slog.Debug("Load AVG", "load", idle.SysinfoAvg())
	slog.Debug("Idle time", "idle", idleTime)
	slog.Debug("Last work was at:", "date", d.lastTriggerTime)

	// Check if the conditions for triggering work are met
	if (idleTime >= idleThreshold && time.Since(d.lastTriggerTime) >= triggerInterval) || time.UnixMicro(0) == d.lastTriggerTime {

		// Wait for any unfinished jobs before scheduling new ones
		d.waitJobs()
		// Update the last trigger time to the current time
		d.lastTriggerTime = time.Now()
		slog.Info("Start addIndexers job", "time", d.lastTriggerTime, d.store)
		// Start the addIndexers job
		d.addIndexers()
	}
}

// addWatcher is responsible for adding a watcher to the specified indexer and managing its lifecycle.
func (d *Daemon) addWatcher(idx indexer.Indexer) {
	defer d.wg.Done()

	// Start the watcher for the indexer
	idx.Watch()
}

// addToIndex is responsible for adding items to the index using the specified indexer.
func (d *Daemon) addToIndex(idx indexer.Indexer) {
	defer d.wg.Done()

	slog.Info("Starting updateIndex", "config", idx.Config())
	// Attempt to update the index using the specified indexer
	td, err := idx.UpdateIndex()
	if err != nil {
		slog.Error("new index failed", "error", err, "indexer", idx)
	}
	slog.Info("Finished updateIndex", "duration", td, "config", idx.Config())
}

// addIndexers creates and starts indexers and watchers for each configuration in idxc.
// It launches goroutines to update the index and watch for changes in the background.
func (d *Daemon) addIndexers() {

	slog.Debug("Indexers", "idx count", len(d.config.Indexer))

	// Iterate over each indexer configuration
	for _, idxConfig := range d.config.Indexer {
		// Create indexers based on the configuration
		for _, idx := range indexer.NewIndexers(d.cancelContext, idxConfig, d.store) {

			// Launch a goroutine to update the indexs
			d.wg.Add(1)
			go d.addToIndex(idx)

			// Launch a goroutine to watch for changes
			d.wg.Add(1)
			go d.addWatcher(idx)
		}
	}
}

// resetScheduler resets the scheduler by updating the ticker interval and resetting the lastTriggerTime.
// It takes an integer parameter 'in' representing the new interval in seconds.
func (d *Daemon) resetScheduler(in int) {
	slog.Info("Scheduler reset", "interval", in)
	// Reset the ticker to the new interval
	d.ticker.Reset(time.Second * time.Duration(in))
	// Reset the last background run time
	d.lastTriggerTime = time.UnixMicro(0)
}

// getExitCode calculates the exit code based on the received OS signal.
// It adds 128 to the signal value to create a unique exit code for each signal.
// The resulting exit code is suitable for conveying the reason for program termination.
func (d *Daemon) getExitCode(sig os.Signal) int {
	return 128 + int(sig.(unix.Signal))
}

// handleHUP handles the SIGHUP signal, typically used for reloading configurations.
// It initiates a graceful shutdown of the daemon, closes the store, performs necessary cleanup,
// and then restarts the daemon with the updated configuration.
// If successful, it returns handled as true, exit as 0, and an error if any occurred.
func (d *Daemon) handleHUP(sig os.Signal) (handled bool, exit int, err error) {
	slog.Info("Reloading...")
	handled = false
	exit = d.getExitCode(sig)

	// Gracefully shut down the daemon
	d.ShutDown()

	// Close the store
	if err = d.store.Close(); err != nil {
		return
	}

	// Capture memory profile
	memprofile()

	// Restart the daemon with the updated configuration
	if d.config, d.store, err = loadUp(); err != nil {
		return
	}

	d.cancelContext, d.cancelJobs = context.WithCancel(context.Background())

	// Reset the scheduler with the new interval
	d.resetScheduler(d.config.Interval)

	// Start the gRPC server in a new goroutine
	go d.grpcServer.Start()

	handled = true
	exit = 0
	slog.Info("Reload complete")
	return
}

// handleQuit handles termination signals (SIGINT, SIGTERM, SIGQUIT).
func (d *Daemon) handleQuit(sig os.Signal) (bool, int, error) {
	// exit := getExitCode(sig)
	exit := 0
	// Return a flag indicating the signal was handled, the exit code, and an error message
	return true, exit, fmt.Errorf("got signal: %v", sig)
}

// handleSignal handles OS signals by determining the appropriate action,
// and returning a flag indicating if the signal was handled, the exit code, and an error message if any.
func (d *Daemon) handleSginal(sig os.Signal) (bool, int, error) {
	slog.Info("Got signal", "signal", sig)
	// Get the exit code associated with the signal
	exit := d.getExitCode(sig)

	// Handle different signals
	switch sig {
	case unix.SIGINT, unix.SIGTERM, unix.SIGQUIT:
		return d.handleQuit(sig)
	case unix.SIGHUP:
		return d.handleHUP(sig)
	default:
		return false, exit, fmt.Errorf("signal %v not handled", sig)
	}
}

// watchSignals sets up a channel for notifying the program of OS signals.
// It creates a buffered channel, registers it to receive specified signals (SIGINT, SIGTERM, SIGQUIT, SIGHUP),
// and returns the channel for signal notifications.
func (d *Daemon) watchSignals() chan os.Signal {
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
