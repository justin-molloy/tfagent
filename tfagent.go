package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-yaml"
	"github.com/justin-molloy/tfagent/logdata"
)

var configFile string = "config.yaml"

type ConfigData struct {
	LogDest  string        `yaml:"logdest"`
	LogLevel string        `yaml:"loglevel"`
	Configs  []ConfigEntry `yaml:"configs"`
}

type ConfigEntry struct {
	Name            string `yaml:"name"`
	SourceDirectory string `yaml:"source_directory"`
	Destination     string `yaml:"destination"`
	Delay           int    `yaml:"delay"`
}

func main() {

	// Read yaml config into ConfigData

	yamlConfig, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("Can't read configuration file!!")
		panic(err)
	}

	var cfg ConfigData
	if err := yaml.Unmarshal(yamlConfig, &cfg); err != nil {
		log.Fatal(err)
	}

	// Configure logging and output config file to log

	logFileName, fileHandle, err := logdata.ConfigLogger(cfg.LogDest)
	if err != nil {
		log.Fatalf("Error configuring logger: %v", err)
	}

	defer fileHandle.Close()
	fmt.Println("Program started. Future log messages written to file:", logFileName)

	prettyYaml, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	logdata.Info(string(prettyYaml))

	// start main routine

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logdata.Error(err.Error())
	}

	defer watcher.Close()

	//	for _, entry := range cfg.Configs {
	//		fmt.Println("Name:", entry.Name)
	//		fmt.Println("Source:", entry.SourceDirectory)
	//		fmt.Println("Destination:", entry.Destination)
	//		fmt.Println("Delay:", entry.Delay)
	//		fmt.Println("---")
	//	}

	// Add source directories to watch from config

	for _, entry := range cfg.Configs {
		fmt.Println("Source:", entry.SourceDirectory)

		err = watcher.Add(entry.SourceDirectory)
		if err != nil {
			logdata.Error(err.Error())
		} else {
			logdata.Info("Added Source Dir" + entry.SourceDirectory)
		}
	}

	// Run function in background

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logdata.Info(fmt.Sprintf("%+v", event))

				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("Modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logdata.Error(err.Error())
			}
		}
	}()

	// Block and create a channel to receive OS signals for interrupts
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	logdata.Info("Program terminated by user: " + sig.String())
	os.Exit(0)

}
