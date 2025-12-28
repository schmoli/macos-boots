package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/installer"
)

var verbose bool

func printBanner() {
	// Gradient styles: cyan -> blue
	line1Style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D9FF"))
	line2Style := lipgloss.NewStyle().Foreground(lipgloss.Color("#0088FF"))
	line3Style := lipgloss.NewStyle().Foreground(lipgloss.Color("#0066FF"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	fmt.Println()
	fmt.Println(line1Style.Render("❯ boots"))
	fmt.Println(line1Style.Render("  ┏┓ ┏━┓┏━┓╺┳╸┏━┓"))
	fmt.Println(line2Style.Render("  ┣┻┓┃ ┃┃ ┃ ┃ ┗━┓"))
	fmt.Println(line3Style.Render("  ┗━┛┗━┛┗━┛ ╹ ┗━┛"))
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  macOS bootstrapper"))
	fmt.Println()
}

func main() {
	// Parse flags and command
	var args []string
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else {
			args = append(args, arg)
		}
	}

	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	// Show banner
	printBanner()

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
	case "docker":
		runErr = runInstall(cfg, "docker")
	case "git":
		runErr = runInstall(cfg, "git")
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
	packagesDir := filepath.Join(home, ".config", "boots", "repo", "packages")
	return config.Load(packagesDir)
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

	result, err := installer.Install(apps, verbose)
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
	fmt.Println("boots - macOS bootstrapper")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  boots              Show status")
	fmt.Println("  boots all          Install all apps")
	fmt.Println("  boots cli          Install CLI tools")
	fmt.Println("  boots apps         Install desktop apps")
	fmt.Println("  boots docker       Install docker tools")
	fmt.Println("  boots git          Install git tools")
	fmt.Println("  boots mas          Install App Store apps")
	fmt.Println("  boots update       Upgrade tracked apps")
	fmt.Println("  boots help         Show this help")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -v, --verbose    Show command details on failure")
}
