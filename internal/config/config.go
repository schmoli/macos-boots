package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Apps map[string]App `yaml:"apps"`
}

type App struct {
	Install     string     `yaml:"install"`      // brew, cask, mas, npm, shell
	Category    string     `yaml:"category"`     // cli, apps, dev, ai, etc.
	Tier        string     `yaml:"tier"`         // auto, interactive
	Description string     `yaml:"description"`  // shown in TUI
	Package     string     `yaml:"package"`      // package name if different from app name
	ID          int        `yaml:"id"`           // App Store ID (mas only)
	Config      *AppConfig `yaml:"config"`       // config files to symlink
	Zsh         string     `yaml:"zsh"`          // zsh module to source
	PostInstall []string   `yaml:"post_install"` // commands to run after
	Depends     []string   `yaml:"depends"`      // dependencies to install first
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

// FilterByCategory returns apps matching the given category
func (c *Config) FilterByCategory(category string) map[string]App {
	result := make(map[string]App)
	for name, app := range c.Apps {
		if app.Category == category {
			result[name] = app
		}
	}
	return result
}

// FilterByInstallType returns apps matching brew or cask
func (c *Config) FilterByInstallType(types ...string) map[string]App {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	result := make(map[string]App)
	for name, app := range c.Apps {
		if typeSet[app.Install] {
			result[name] = app
		}
	}
	return result
}
