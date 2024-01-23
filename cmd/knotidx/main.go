package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

const (
	programName string = "knotidx"
)

var (
	version = "development"
	commit  string
	date    = time.Now().String()

	programErr      error
	programExitCode      = 1                  // Exit code set to 1 by default
	programLevel         = new(slog.LevelVar) // Info by default
	programProfiler bool = false

	daemonCmd      = flag.Bool("daemon", false, "run knotidx daemon")
	showConfigCmd  = flag.Bool("show-config", false, "show knotidx config")
	checkConfigCmd = flag.Bool("check-config", false, "check knotidx config for errors")
	searchCmd      = flag.Bool("search", false, "interactive index search")
)

func main() {

	// flag.CommandLine.""
	// flag.CommandLine.Set("alsologtostderr", "false")
	// pflag.CommandLine.MarkHidden("log-backtrace-at")
	// pflag.CommandLine.MarkHidden("log-dir")
	// pflag.CommandLine.MarkHidden("logtostderr")
	// pflag.CommandLine.MarkHidden("log-file")          //nolint:errcheck
	// pflag.CommandLine.MarkHidden("log-file-max-size") //nolint:errcheck
	// pflag.CommandLine.MarkHidden("one-output")        //nolint:errcheck
	// pflag.CommandLine.MarkHidden("skip-log-headers")  //nolint:errcheck
	// pflag.CommandLine.MarkHidden("stderrthreshold")
	// pflag.CommandLine.MarkHidden("vmodule")

	flag.Parse()
	// Set slog logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: programLevel,
	})).With(
		slog.String("program", programName),
		slog.Int("pid", os.Getpid()),
	))
	programLevel.Set(slog.LevelDebug)

	slog.Info("build info", "version", version, "commit", commit, "date", date)
	if *daemonCmd {
		daemon()
		return
	}
	if *showConfigCmd {
		showConfig()
		return
	}

	if *checkConfigCmd {
		checkConfig()
		return
	}

	if *searchCmd {
		// Load configs and open store
		if gConf, gStore, programErr = startUp(); programErr != nil {
			return
		}
		defer shutDown()
		Search()
		return
	}
}

// Do program shutdown
func shutDown() {
	slog.Info("Stoopping knotidx")

	if *daemonCmd {
		if ticker != nil {
			slog.Debug("Stoopping ticker")
			ticker.Stop()
		}
		slog.Info("Waiting for all indexers to finish")
		// Close watchers channel
		if quitWtCh != nil {
			close(quitWtCh)
		}
		wg.Wait()
	}

	// Close the store
	gStore.Close()

	if programErr != nil && programExitCode != 0 {
		slog.Error("exit", "error", programErr)
	}
	memprofile()
	slog.Info("knotidx stopped", "exit", programExitCode)
	os.Exit(programExitCode)
}

// Startup seq reloading configs and create/open store
func startUp() (conf config.Config, s store.Store, err error) {
	// Load default config and file config
	if conf, err = reloadConfig(); err != nil {
		return
	}
	// Open the store
	if s, err = newGlobalStore(conf.Store); err != nil {
		return
	}
	return
}

// Check config for errors
func checkConfig() {
	_, err := reloadConfig()
	if err != nil {
		slog.Error("Config check error")
		os.Exit(1)
	} else {
		slog.Info("Config check success")
		os.Exit(0)
	}
}

// Ouput current config
func showConfig() {
	conf, _ := reloadConfig()
	slog.Info("knotidx config", "config", conf)
}

// (Re)Load default config and file config
func reloadConfig() (conf config.Config, err error) {
	if conf, err = config.DefaultConfig().Load(); err != nil {
		slog.Error("Can't read config from toml files", "error", err)
		return
	}
	return
}
