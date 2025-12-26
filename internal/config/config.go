package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Apps map[string]App `yaml:"apps"`
}

type App struct {
	Install     string   `yaml:"install"`     // brew, cask, mas, script
	Category    string   `yaml:"category"`    // cli, apps, dev, etc.
	Tier        string   `yaml:"tier"`        // auto, interactive
	Description string   `yaml:"description"` // shown in TUI
	ID          int      `yaml:"id"`          // App Store ID (mas only)
	Config      *AppConfig `yaml:"config"`    // config files to symlink
	Zsh         string   `yaml:"zsh"`         // zsh module to source
	PostInstall []string `yaml:"post_install"` // commands to run after
}

type AppConfig struct {
	Source string `yaml:"source"` // path in repo
	Dest   string `yaml:"dest"`   // path in home
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// AppsByCategory returns apps grouped by category
func (c *Config) AppsByCategory() map[string][]string {
	result := make(map[string][]string)
	for name, app := range c.Apps {
		result[app.Category] = append(result[app.Category], name)
	}
	return result
}

// AppsByTier returns apps grouped by tier
func (c *Config) AppsByTier() map[string][]string {
	result := make(map[string][]string)
	for name, app := range c.Apps {
		result[app.Tier] = append(result[app.Tier], name)
	}
	return result
}
