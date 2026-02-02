package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Transport string          `yaml:"transport"` // "stdio" or "sse"
	Port      int             `yaml:"port"`
	Security  SecurityConfig  `yaml:"security"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type SecurityConfig struct {
	AllowedPaths        []string `yaml:"allowed_paths"`
	BlockedCommands     []string `yaml:"blocked_commands"`
	RequireConfirmation []string `yaml:"require_confirmation"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

func DefaultConfig() *Config {
	return &Config{
		Transport: "stdio",
		Port:      8686,
		Security: SecurityConfig{
			AllowedPaths:        []string{},
			BlockedCommands:     []string{"rm -rf /", "mkfs", "dd if="},
			RequireConfirmation: []string{},
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "/tmp/lingti-bot.log",
		},
	}
}

func ConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Preferences", "Lingti")
	case "linux":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "lingti")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".lingti")
	}
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "bot.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}
