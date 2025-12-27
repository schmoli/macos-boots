package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/installer"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// Handle help without loading config
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		printHelp()
		return
	}

	// Auto-pull on any command (except help)
	if installer.AutoPull() {
		fmt.Println()
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	var runErr error
	switch cmd {
	case "":
		installer.Status(cfg)
	case "all":
		runErr = runInstall(cfg, "")
	case "cli":
		runErr = runInstall(cfg, "cli")
	case "apps":
		runErr = runInstall(cfg, "apps")
	case "mas":
		runErr = runInstall(cfg, "mas")
	case "update":
		runErr = installer.Upgrade(cfg)
	case "status":
		installer.Status(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
		os.Exit(1)
	}

	// Check if zshrc was modified
	if installer.CheckZshrcModified() {
		fmt.Println()
		installer.LogWarn("Run: source ~/.zshrc")
	}
}

func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")
	return config.Load(configPath)
}

func runInstall(cfg *config.Config, category string) error {
	apps := cfg.Apps
	if category != "" {
		apps = cfg.FilterByCategory(category)
	}

	if len(apps) == 0 {
		fmt.Printf("No apps found for category: %s\n", category)
		return nil
	}

	result, err := installer.Install(apps, true)
	if err != nil {
		return err
	}

	// Print summary
	fmt.Println()
	if len(result.Installed) > 0 {
		installer.LogSuccess(fmt.Sprintf("Installed: %v", result.Installed))
	}
	if len(result.Failed) > 0 {
		installer.LogFail(fmt.Sprintf("Failed: %v", result.Failed))
	}
	if len(result.Installed) == 0 && len(result.Failed) == 0 {
		label := "tools"
		if category != "" {
			label = category + " tools"
		}
		installer.LogSuccess(fmt.Sprintf("All %s installed", label))
	}

	return nil
}

func printHelp() {
	fmt.Println("macos-setup - fast macOS setup")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  macos-setup          Show status")
	fmt.Println("  macos-setup all      Install all apps")
	fmt.Println("  macos-setup cli      Install CLI tools")
	fmt.Println("  macos-setup apps     Install desktop apps")
	fmt.Println("  macos-setup mas      Install App Store apps")
	fmt.Println("  macos-setup update   Upgrade tracked apps")
	fmt.Println("  macos-setup help     Show this help")
}
