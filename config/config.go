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
	Transfers    []ConfigEntry `yaml:"transfers"`
}

type ConfigEntry struct {
	Name            string `yaml:"name"`
	SourceDirectory string `yaml:"source_directory"`
	RemotePath      string `yaml:"remotepath"`
	Delay           *int   `yaml:"delay"`
	Streaming       *bool  `yaml:"streaming"`
	TransferType    string `yaml:"transfertype"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Server          string `yaml:"server"`
	Port            string `yaml:"port"`
}

type FlagOptions struct {
	LogFile      string
	ConfigFile   string
	LogLevel     string
	LogToConsole bool
}

func (c *ConfigData) SetDefaults() {
	defaultDelay := 2
	defaultStreaming := false
	for i := range c.Transfers {
		if c.Transfers[i].Delay == nil {
			slog.Warn("Default delay for " + c.Transfers[i].Name + " not set. Default is " + strconv.Itoa(defaultDelay))
			c.Transfers[i].Delay = &defaultDelay
		}
		if c.Transfers[i].Streaming == nil {
			slog.Warn("Streaming value for " + c.Transfers[i].Name + " not set. Default is " + strconv.FormatBool(defaultStreaming))
			c.Transfers[i].Streaming = &defaultStreaming
		}

	}
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

	// ensure all transfer entries have a value before returning.
	cfg.SetDefaults()

	return cfg, err
}

func ParseFlags() FlagOptions {
	var flags FlagOptions

	flag.StringVar(&flags.LogFile, "logfile", "logs/app.log", "Path to log file")
	flag.StringVar(&flags.ConfigFile, "config", "config.yaml", "Path to config file. If this is not set, config is read from current directory")
	flag.StringVar(&flags.LogLevel, "loglevel", "info", "Log level (debug, info, warn, error)")
	flag.BoolVar(&flags.LogToConsole, "console", true, "Log to consle instead of file")

	flag.Parse()
	return flags
}

func SetupLogger(logFilePath string, level string, logToConsole bool) (*os.File, error) {
	var output *os.File
	var err error

	if !logToConsole {
		output, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	} else {
		output = os.Stdout
	}

	// Convert log level string to slog.Level
	var slogLevel slog.Level
	switch strings.ToLower(level) {
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
	if logToConsole {
		slog.Info("Program started. Log messages output to stdout.")
		return nil, nil
	}

	slog.Info("Program started. Future log messages will be written here.", "path", logFilePath)
	return output, nil
}
