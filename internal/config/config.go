package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Apps map[string]App
}

type App struct {
	Install     string     `yaml:"install"`
	Category    string     `yaml:"-"` // inferred from path
	Description string     `yaml:"description"`
	Package     string     `yaml:"package"`
	ID          int        `yaml:"id"`
	Config      *AppConfig `yaml:"config"`
	PostInstall []string   `yaml:"post_install"`
	Depends     []string   `yaml:"depends"`
	Init        bool       `yaml:"init"` // marks app as init/base tool
}

type AppConfig struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

// Load scans packages/<category>/<name>/app.yaml files
func Load(appsDir string) (*Config, error) {
	cfg := &Config{Apps: make(map[string]App)}

	// Scan category directories
	categories, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if !cat.IsDir() {
			continue
		}
		catName := cat.Name()
		catPath := filepath.Join(appsDir, catName)

		// Scan app directories within category
		apps, err := os.ReadDir(catPath)
		if err != nil {
			continue
		}

		for _, appDir := range apps {
			if !appDir.IsDir() {
				continue
			}
			appName := appDir.Name()
			appYaml := filepath.Join(catPath, appName, "app.yaml")

			data, err := os.ReadFile(appYaml)
			if err != nil {
				continue // skip if no app.yaml
			}

			var app App
			if err := yaml.Unmarshal(data, &app); err != nil {
				continue
			}

			app.Category = catName
			cfg.Apps[appName] = app
		}
	}

	return cfg, nil
}

// LoadLegacy loads from single apps.yaml (for migration)
func LoadLegacy(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Apps map[string]App `yaml:"apps"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return &Config{Apps: wrapper.Apps}, nil
}

// HasInitZsh checks if app has init.zsh in repo
func HasInitZsh(appsDir, category, name string) bool {
	initPath := filepath.Join(appsDir, category, name, "init.zsh")
	_, err := os.Stat(initPath)
	return err == nil
}

// AppsByCategory returns apps grouped by category
func (c *Config) AppsByCategory() map[string][]string {
	result := make(map[string][]string)
	for name, app := range c.Apps {
		result[app.Category] = append(result[app.Category], name)
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

// FilterByInstallType returns apps matching install types
func (c *Config) FilterByInstallType(types ...string) map[string]App {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[strings.ToLower(t)] = true
	}
	result := make(map[string]App)
	for name, app := range c.Apps {
		if typeSet[strings.ToLower(app.Install)] {
			result[name] = app
		}
	}
	return result
}

// InitApps returns all apps marked with init: true
func (c *Config) InitApps() map[string]App {
	result := make(map[string]App)
	for name, app := range c.Apps {
		if app.Init {
			result[name] = app
		}
	}
	return result
}
