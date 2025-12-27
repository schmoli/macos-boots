package state

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type State struct {
	Installed map[string]string `yaml:"installed"`
}

func statePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "macos-setup", "state.yaml")
}

func Load() (*State, error) {
	s := &State{Installed: make(map[string]string)}

	data, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, nil // ignore read errors, return empty
	}

	if err := yaml.Unmarshal(data, s); err != nil {
		return s, nil // ignore parse errors, return empty
	}

	if s.Installed == nil {
		s.Installed = make(map[string]string)
	}

	return s, nil
}

func (s *State) Save() error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	path := statePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Atomic write via temp file
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func (s *State) MarkInstalled(name string) {
	s.Installed[name] = time.Now().Format("2006-01-02")
	s.Save() // best effort
}

func (s *State) MarkRemoved(name string) {
	delete(s.Installed, name)
	s.Save() // best effort
}

func (s *State) IsTracked(name string) bool {
	_, ok := s.Installed[name]
	return ok
}
