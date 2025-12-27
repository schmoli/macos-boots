# CLI Tools Init Audit

Generated: 2025-12-27

## Tools Needing Setup (11)

### fzf
**Priority: High** - Key productivity tool

- [ ] Init: `source <(fzf --zsh)`

Provides: CTRL-R (history), CTRL-T (file paste), ALT-C (cd), `**<TAB>` fuzzy completion

---

### bat
- [ ] Init: `eval "$(bat --completion zsh)"`
- [ ] Aliases:
  ```zsh
  alias cat='bat --paging=never'
  export MANPAGER="bat -plman"
  ```

---

### eza
- [ ] Aliases:
  ```zsh
  alias ls='eza'
  alias ll='eza -l --header --icons'
  alias la='eza -la --header --icons'
  alias lt='eza --tree'
  ```

Note: Icons require Nerd Font (you have font-jetbrains-mono-nerd-font installed)

---

### uv
- [ ] Init: `eval "$(uv generate-shell-completion zsh)"`

Optional: Oh My Zsh `uv` plugin provides 24 aliases (uva, uvrm, uvs, etc.)

---

### gh
- [ ] Completions: `gh completion -s zsh > "${fpath[1]}/_gh"`

Or simpler: `eval "$(gh completion -s zsh)"` in .zshrc

---

### kind
- [ ] Init: `source <(kind completion zsh)`

---

### helm
- [ ] Completions: `helm completion zsh > "${fpath[1]}/_helm"`

Or: add `helm` to Oh My Zsh plugins for auto-managed completions + aliases

---

### yq
- [ ] Completions: `yq completion zsh > "${fpath[1]}/_yq"`

---

### glow
- [ ] Completions: `glow completion zsh > "${fpath[1]}/_glow"`

---

### ripgrep
- [ ] Completions: may need fpath config if brew completions not working
- [ ] Optional: `--smart-case` commonly aliased or added to RIPGREP_CONFIG_PATH

Note: Watch for Oh My Zsh Rails plugin conflict (aliases `rg` to `rails generate`)

---

### fd
- [ ] Ensure `eval "$(brew shellenv)"` in .zprofile (for FPATH)

Note: Oh My Zsh has conflicting `fd` alias (finds directories) - may need to disable

---

## Already Complete (7)

| Tool | Notes |
|------|-------|
| jq | zsh has built-in completions |
| btop | Standalone TUI, no shell integration |
| htop | Standalone TUI, no shell integration |
| tmux | brew handles completions via FPATH |
| watch | Standard Unix utility, no setup |
| mas | Works out of box (no zsh completions exist) |
| gitui | Standalone TUI, no shell integration |

## Already Have init.zsh (3)

| Tool | Current Setup |
|------|---------------|
| zoxide | `eval "$(zoxide init zsh)"` |
| fnm | `eval "$(fnm env --use-on-cd)"` |
| yazi | `alias yz=yazi` |

---

## Recommended Priority

1. **fzf** - Huge productivity boost (history search, fuzzy finding)
2. **eza** - Daily `ls` replacement
3. **bat** - Better cat/man paging
4. **uv** - Python dev completions
5. **gh** - GitHub workflow completions
6. Rest as needed
