case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) export PATH="$HOME/.local/bin:$PATH" ;; esac
[ -x /opt/homebrew/bin/brew ] && eval "$(/opt/homebrew/bin/brew shellenv)"
[ -x /home/linuxbrew/.linuxbrew/bin/brew ] && eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
command -v mise >/dev/null && eval "$(mise activate zsh)"
command -v zoxide >/dev/null && eval "$(zoxide init zsh)"
setopt autocd

export HISTSIZE=10000000
export SAVEHIST=10000000

alias glo='git log --oneline'
alias repo='gh repo view --web'
alias pr='tb pr'
alias commit='tb commit'
alias checkout='tb checkout'
alias hot='tb hot'
