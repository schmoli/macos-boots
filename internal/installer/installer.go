package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/schmoli/macos-setup/internal/config"
)

// InstalledBrewPackages returns set of installed brew/cask packages
func InstalledBrewPackages() map[string]bool {
	installed := make(map[string]bool)

	// Get brew formulae
	cmd := exec.Command("/opt/homebrew/bin/brew", "list", "--formula", "-1")
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				installed[line] = true
			}
		}
	}

	// Get casks
	cmd = exec.Command("/opt/homebrew/bin/brew", "list", "--cask", "-1")
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				installed[line] = true
			}
		}
	}

	return installed
}

// GenerateBrewfile creates a temp Brewfile for the given apps
func GenerateBrewfile(apps map[string]config.App) (string, error) {
	var lines []string

	for name, app := range apps {
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		switch app.Install {
		case "brew":
			lines = append(lines, fmt.Sprintf("brew \"%s\"", pkg))
		case "cask":
			lines = append(lines, fmt.Sprintf("cask \"%s\"", pkg))
		}
	}

	if len(lines) == 0 {
		return "", nil
	}

	tmpFile := "/tmp/macos-setup-Brewfile"
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return "", err
	}

	return tmpFile, nil
}
