# bat - cat with syntax highlighting
eval "$(bat --completion zsh)"

alias cat='bat --paging=never'
export MANPAGER="bat -plman"
