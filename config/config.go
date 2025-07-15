package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"
)

type ConfigData struct {
	LogDest   string        `yaml:"logdest"`
	LogLevel  string        `yaml:"loglevel"`
	Transfers []ConfigEntry `yaml:"transfers"`
}

type ConfigEntry struct {
	Name            string `yaml:"name"`
	SourceDirectory string `yaml:"source_directory"`
	Destination     string `yaml:"destination"`
	Delay           *int   `yaml:"delay"`
	Streaming       *bool  `yaml:"streaming"`
}

func (c *ConfigData) SetDefaults() {
	defaultDelay := 2
	defaultStreaming := false
	for i := range c.Transfers {
		if c.Transfers[i].Delay == nil {
			slog.Info("Default delay for " + c.Transfers[i].Name + " not set. Default is " + strconv.Itoa(defaultDelay))
			c.Transfers[i].Delay = &defaultDelay
		}
		if c.Transfers[i].Streaming == nil {
			slog.Info("Streaming value for " + c.Transfers[i].Name + " not set. Default is " + strconv.FormatBool(defaultStreaming))
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

	// Log the loaded config
	//	prettyYaml, err := yaml.Marshal(cfg)
	//	if err != nil {
	//		return ConfigData{}, nil, fmt.Errorf("could not marshal config for logging: %w", err)
	//	}
	//	logdata.Info(string(prettyYaml))

	return cfg, err
}
