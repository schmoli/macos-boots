package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update":
			if err := runUpdate(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "install":
			if err := runInstallAll(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "--help", "-h":
			printHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("macos-setup - TUI for setting up macOS")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  macos-setup          Launch TUI")
	fmt.Println("  macos-setup install  Install all apps directly")
	fmt.Println("  macos-setup update   Pull latest and rebuild")
	fmt.Println("  macos-setup help     Show this help")
}

func runUpdate() error {
	home, _ := os.UserHomeDir()
	repoDir := filepath.Join(home, ".config", "macos-setup", "repo")
	binPath := filepath.Join(repoDir, "bin", "macos-setup")

	// Ensure Homebrew is in PATH for go commands
	brewPrefix := "/opt/homebrew"
	path := os.Getenv("PATH")
	if _, err := os.Stat(brewPrefix + "/bin/brew"); err == nil {
		os.Setenv("PATH", brewPrefix+"/bin:"+brewPrefix+"/sbin:"+path)
	}

	fmt.Println("Checking for updates...")

	// Fetch and compare commits
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repoDir
	cmd.Run()

	localCmd := exec.Command("git", "rev-parse", "HEAD")
	localCmd.Dir = repoDir
	localHash, _ := localCmd.Output()

	remoteCmd := exec.Command("git", "rev-parse", "origin/main")
	remoteCmd.Dir = repoDir
	remoteHash, _ := remoteCmd.Output()

	if string(localHash) == string(remoteHash) {
		fmt.Println("✓ Already up to date!")
		return nil
	}

	fmt.Println()

	// Git pull
	fmt.Println("→ Pulling latest...")
	cmd = exec.Command("git", "pull", "--rebase")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Remove old binary to force rebuild
	os.Remove(binPath)

	// Rebuild
	fmt.Println("→ Rebuilding...")
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = repoDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	cmd = exec.Command("go", "build", "-o", binPath, "./cmd/macos-setup/")
	cmd.Dir = repoDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Updated!")
	return nil
}

func isAppInstalled(name string, app config.App) bool {
	var cmd *exec.Cmd
	switch app.Install {
	case "brew":
		cmd = exec.Command("/opt/homebrew/bin/brew", "list", name)
	case "cask":
		cmd = exec.Command("/opt/homebrew/bin/brew", "list", "--cask", name)
	case "npm":
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}
		cmd = exec.Command("npm", "list", "-g", pkg)
	case "shell":
		home, _ := os.UserHomeDir()
		appDir := filepath.Join(home, ".config", "macos-setup", "apps", name)
		_, err := os.Stat(appDir)
		return err == nil
	default:
		return false
	}
	return cmd.Run() == nil
}

func runInstallAll() error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Installing all apps...")
	fmt.Println()

	var installed, failed []string

	for name, app := range cfg.Apps {
		// Skip required tier (already installed at startup)
		if app.Tier == "required" {
			continue
		}

		// Skip already installed
		if isAppInstalled(name, app) {
			fmt.Printf("✓ %s (already installed)\n", name)
			continue
		}

		fmt.Printf("→ Installing %s...\n", name)

		var cmd *exec.Cmd
		switch app.Install {
		case "brew":
			cmd = exec.Command("/opt/homebrew/bin/brew", "install", name)
		case "cask":
			cmd = exec.Command("/opt/homebrew/bin/brew", "install", "--cask", name)
		case "npm":
			pkg := name
			if app.Package != "" {
				pkg = app.Package
			}
			cmd = exec.Command("npm", "install", "-g", pkg)
		case "mas":
			cmd = exec.Command("mas", "install", fmt.Sprintf("%d", app.ID))
		case "shell":
			// Shell-only, just add zsh integration
			cmd = nil
		default:
			fmt.Printf("  ⚠ Unknown install type: %s\n", app.Install)
			continue
		}

		if cmd != nil {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("  ✗ Failed\n")
				failed = append(failed, name)
				continue
			}
		}

		// Add zsh integration if defined
		if app.Zsh != "" {
			zshDir := filepath.Join(home, ".config", "macos-setup", "apps", name)
			os.MkdirAll(zshDir, 0755)
			zshPath := filepath.Join(zshDir, "zshrc.zsh")
			os.WriteFile(zshPath, []byte(app.Zsh), 0644)
		}

		// Run post_install
		for _, postCmd := range app.PostInstall {
			fmt.Printf("  → %s\n", postCmd)
			c := exec.Command("zsh", "-c", postCmd)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Run()
		}

		fmt.Printf("  ✓ Done\n")
		installed = append(installed, name)
	}

	fmt.Println()
	fmt.Printf("✓ Installed %d apps", len(installed))
	if len(failed) > 0 {
		fmt.Printf(", %d failed", len(failed))
	}
	fmt.Println()

	return nil
}
