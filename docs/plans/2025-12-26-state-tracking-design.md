# State Tracking Design

Track apps installed via macos-setup for history/debugging.

## Storage

File: `~/.config/macos-setup/state.yaml`

```yaml
installed:
  jq: 2025-01-15
  fzf: 2025-01-15
```

## Package: internal/state

```go
type State struct {
    Installed map[string]string `yaml:"installed"` // app -> date
}

func Load() (*State, error)
func (s *State) Save() error
func (s *State) MarkInstalled(name string)
func (s *State) MarkRemoved(name string)
```

## Behavior

- File created on first install
- Missing file = empty state
- Atomic writes (temp file + rename)
- Best effort - never blocks installs

## Integration

- TUI: mark on successful install/remove
- CLI install: mark on successful install
- State is "we installed it", not "it exists now"

## Not Doing

- No TUI indicator for "installed by us"
- No reconciliation command
- No tracking pre-existing apps
