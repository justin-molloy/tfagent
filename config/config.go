package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

type ConfigData struct {
	LogFile      string        `yaml:"logdest"`
	LogLevel     string        `yaml:"loglevel"`
	LogToConsole bool          `yaml:"logtoconsole"`
	Delay        int           `yaml:"delay"`
	Transfers    []ConfigEntry `yaml:"transfers"`
}

type ConfigEntry struct {
	Name            string `yaml:"name"`
	SourceDirectory string `yaml:"source_directory"`
	RemotePath      string `yaml:"remotepath"`
	Streaming       *bool  `yaml:"streaming"`
	TransferType    string `yaml:"transfertype"`
	Username        string `yaml:"username"`
	PrivateKey      string `yaml:"privatekey"`
	Password        string `yaml:"password"`
	Server          string `yaml:"server"`
	Port            string `yaml:"port"`
	Filter          string `yaml:"filter"`
	ArchiveDest     string `yaml:"archive_dest"`
	ActionOnSuccess string `yaml:"action_on_success"`
}

type FlagOptions struct {
	LogFile      string
	ConfigFile   string
	LogLevel     string
	LogToConsole bool
	PrtConf      bool
}

// ParseFlags sets some of the defaults for the program - eg config.yaml is expected to be in the
// current directory by default.

func ParseFlags() FlagOptions {
	var flags FlagOptions

	flag.StringVar(&flags.LogFile, "logfile", "logs/app.log", "Path to log file")
	flag.StringVar(&flags.ConfigFile, "config", "config.yaml", "Path to config file. If this is not set, config is read from current directory")
	flag.StringVar(&flags.LogLevel, "loglevel", "info", "Log level (debug, info, warn, error)")
	flag.BoolVar(&flags.LogToConsole, "console", false, "Log to consle instead of file")
	flag.BoolVar(&flags.PrtConf, "prtconf", false, "Print config and exit")

	flag.Parse()
	return flags
}

func LoadConfig(configFile string) (ConfigData, error) {

	// Read yaml config into ConfigData
	yamlConfig, err := os.ReadFile(configFile)
	if err != nil {
		return ConfigData{}, fmt.Errorf("can't read configuration file: %w", err)
	}

	var cfg ConfigData
	if err := yaml.Unmarshal(yamlConfig, &cfg); err != nil {
		return ConfigData{}, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// This section sets some default values for individual transfers.
	// Here we ensure all transfer entries have a minimum value set before returning.

	defaultStreaming := false
	for i := range cfg.Transfers {
		if cfg.Transfers[i].Streaming == nil {
			slog.Warn("Streaming value for " + cfg.Transfers[i].Name + " not set. Default is " + strconv.FormatBool(defaultStreaming))
			cfg.Transfers[i].Streaming = &defaultStreaming
		}

	}

	return cfg, err
}

func SetupLogger(cfg ConfigData, flags FlagOptions) (*os.File, error) {
	var output *os.File
	var err error

	// Override config with flags if different from defaults

	if cfg.LogFile == "" {
		cfg.LogFile = flags.LogFile
	}
	if flags.LogLevel != "info" {
		cfg.LogLevel = flags.LogLevel
	}
	if flags.LogToConsole {
		cfg.LogToConsole = true
		output = os.Stdout
	} else {
		output, err = os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	// Convert log level string to slog.Level

	var slogLevel slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn", "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo // fallback
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: slogLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Return nil for file if we're using stdout
	if cfg.LogToConsole {
		slog.Info("Program started. Log messages output to stdout.")
		return nil, nil
	}

	slog.Info("Program started. Future log messages will be written here.", "path", cfg.LogFile)
	return output, nil
}

func PrintConfig(cfg ConfigData) {
	yamlData, _ := yaml.Marshal(cfg)
	fmt.Println(string(yamlData))
}
