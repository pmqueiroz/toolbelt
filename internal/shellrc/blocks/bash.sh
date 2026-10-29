[ -x /opt/homebrew/bin/brew ] && eval "$(/opt/homebrew/bin/brew shellenv)"
[ -x /home/linuxbrew/.linuxbrew/bin/brew ] && eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
command -v mise >/dev/null && eval "$(mise activate bash)"
command -v zoxide >/dev/null && eval "$(zoxide init bash)"
shopt -s autocd

export HISTSIZE=
export HISTFILESIZE=

alias glo='git log --oneline'
alias repo='gh repo view --web'
alias pr='tb pr'
alias commit='tb commit'
alias checkout='tb checkout'
alias hot='tb hot'
alias penv='tb penv'
