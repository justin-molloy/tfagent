package main

import (
	"log"
	"log/slog"
	"os"

	// "os/signal"

	"golang.org/x/sys/windows/svc"

	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/processor"
	"github.com/justin-molloy/tfagent/selector"
	"github.com/justin-molloy/tfagent/service"
	"github.com/justin-molloy/tfagent/tracker"
)

func main() {

	// Application name used for Windows service call, and for defining
	// where the config file should be(if installed using installer).

	const AppName = "TFAgent"

	flags := config.ParseFlags()

	// Get and read config file

	configFile, err := config.GetConfigFile(AppName)
	if err != nil {
		log.Fatalf("Can't find where the config file lives: %v", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		os.Exit(1)
	}

	// print config and exit if required before wasting further cycles

	if flags.PrtConf {
		config.PrintConfig(*cfg)
		os.Exit(0)
	}

	// set up logging once we've determined where we should log to and
	// exit if we can't set logging up(if we can't set up logging, there's
	// a bigger problem that needs to be resolved). Note that the commandline
	// flags may change the config.

	logFile, err := config.SetupLogger(cfg, flags)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// trackerMap holds files that are eligible to be processed by the selector routine

	trackerMap := tracker.NewEventTracker()

	// queue for files to be processed

	fileQueue := make(chan string, 100) // buffered to avoid blocking

	// Selector views events that have been added to the tracker map, and
	// moves them to the fileQueue when eligible.

	processingMap := selector.NewFileSelector()

	isService, err := svc.IsWindowsService()
	if err != nil {
		slog.Error("failed to determine session type", "error", err)
		os.Exit(1)
	}

	// Run the service if called from Windows Service, or run standalone if not.
	// Both options use the same routines once started. The tracker watches the
	// filesystem for events, and queues eligible files (determined by the
	// filter in config for each transfer entry)

	if isService {
		slog.Info("Running as Windows Service", "isService", isService)
		svc.Run(AppName, &service.TFAgentService{
			Name:    AppName,
			Config:  cfg,
			Tracker: trackerMap})
		return
	} else {
		slog.Info("Running as standalone app outside of Windows Service Control Manager")
		go tracker.StartTracker(cfg, trackerMap)
		go selector.StartSelector(cfg, trackerMap, fileQueue, processingMap)
		go processor.StartProcessor(cfg, fileQueue, processingMap)
	}

	// keep main alive (may replace this with sync.WaitGroup later)
	select {}

	// Block and create a channel to receive OS signals for interrupts
	//	sigs := make(chan os.Signal, 1)
	//	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	//	sig := <-sigs
	//	slog.Info("Program terminated by user", "signal", sig.String())
	//	os.Exit(0)

}
