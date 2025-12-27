package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
