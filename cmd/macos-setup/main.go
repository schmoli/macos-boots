package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/installer"
)

func main() {
	// Auto-pull on any command
	if installer.AutoPull() {
		fmt.Println()
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var runErr error
	switch cmd {
	case "", "all":
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
	case "help", "--help", "-h":
		printHelp()
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
		fmt.Println("⚠  Run: source ~/.zshrc")
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
		fmt.Printf("✓ Installed: %v\n", result.Installed)
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped (already installed): %v\n", result.Skipped)
	}
	if len(result.Failed) > 0 {
		fmt.Printf("✗ Failed: %v\n", result.Failed)
	}

	return nil
}

func printHelp() {
	fmt.Println("macos-setup - fast macOS setup")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  macos-setup          Install all apps")
	fmt.Println("  macos-setup cli      Install CLI tools")
	fmt.Println("  macos-setup apps     Install desktop apps")
	fmt.Println("  macos-setup mas      Install App Store apps")
	fmt.Println("  macos-setup update   Upgrade tracked apps")
	fmt.Println("  macos-setup status   Show installed vs available")
	fmt.Println("  macos-setup help     Show this help")
}
