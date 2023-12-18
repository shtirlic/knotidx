package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"

	"net/http"
	_ "net/http/pprof"
)

const (
// programName string = "knotidx"
)

var (
	version = "development"
	commit  string
	date    = time.Now().String()

	programName     = "knotidx"
	programErr      error
	programExitCode      = 1                  // Exit code set to 1 by default
	programLevel         = new(slog.LevelVar) // Info by default
	programProfiler bool = false

	daemon *Daemon
	client *Client

	configCmd      = flag.String("config", config.DefaultConfigFile, "knotidx config file (default: knotidx.toml) ")
	daemonCmd      = flag.Bool("daemon", false, "run knotidx daemon")
	showConfigCmd  = flag.Bool("show-config", false, "show knotidx config")
	checkConfigCmd = flag.Bool("check-config", false, "check knotidx config for errors")
	clientCmd      = flag.Bool("client", false, "interactive index search")
	jsonCmd        = flag.Bool("json", false, "json only output")
	debugCmd       = flag.Bool("debug", false, "debug mode")
	versionCmd     = flag.Bool("version", false, "show version")
)

func main() {
	flag.Parse()

	// Set slog logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: programLevel,
	})).With(
		slog.String("program", programName),
	// slog.Int("pid", os.Getpid()),
	))

	if *jsonCmd {
		programLevel.Set(slog.LevelError)
	}

	if *debugCmd {
		programLevel.Set(slog.LevelDebug)
		// programProfiler = true
		go func() {
			runtime.SetBlockProfileRate(1)
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	slog.Info("build info", "version", version, "commit", commit, "date", date)
	if *versionCmd {
		return
	}

	var conf config.Config
	var s store.Store

	// Load configs and open store
	if conf, s, programErr = loadUp(); programErr != nil {
		return
	}
	defer shutDown(s)

	if *showConfigCmd {
		showConfig()
	}

	if *checkConfigCmd {
		checkConfig()
	}

	if *daemonCmd {
		daemon = NewDaemon(conf, s)
		programExitCode, programErr = daemon.Start()
	}

	if *clientCmd {
		client = NewClient(conf.GRPC)
		programExitCode, programErr = client.Start()
	}
}

// Do program shutdown
func shutDown(s store.Store) {
	slog.Info("Stopping knotidx")

	// If was run in daemon mode
	if daemon != nil {
		daemon.ShutDown()
	}

	// Close the store
	if s != nil {
		// TODO: wrap err
		err := s.Close()
		if err != nil {
			slog.Error("err", err)
		}
	}

	if programErr != nil && programExitCode != 0 {
		slog.Error("exit", "error", programErr)
	}
	if programErr == nil {
		programExitCode = 0
	}
	memprofile()

	if r := recover(); r != nil {
		slog.Info("knotidx paniced", "panic", r)
		if *debugCmd {
			panic(r)
		}
		programExitCode = 2
		os.Exit(programExitCode)
	} else {
		slog.Info("knotidx stopped", "exit", programExitCode)
		os.Exit(programExitCode)
	}
}

// Startup seq reloading configs and create/open store
func loadUp() (c config.Config, s store.Store, err error) {
	// Load default config and file config
	if c, err = reloadConfig(); err != nil {
		return
	}
	if *clientCmd {
		return
	}
	s, err = newStore(c.Store)
	return
}

// Check config for errors
func checkConfig() {
	_, err := reloadConfig()
	if err != nil {
		slog.Error("Config check error")
		os.Exit(1)
	} else {
		slog.Info("Config check successful")
		os.Exit(0)
	}
}

// Ouput current config
func showConfig() {
	conf, _ := reloadConfig()
	slog.Info("knotidx config", "config", conf)
}

// (Re)Load default config and file config
func reloadConfig() (config.Config, error) {
	var c config.Config
	var err error
	if c, err = config.DefaultConfig().Load(*configCmd); err != nil {
		slog.Error("Can't read config from toml files", "error", err)
		return config.Config{}, err
	}
	return c, nil
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
