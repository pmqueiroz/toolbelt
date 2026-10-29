if (Get-Command mise -ErrorAction SilentlyContinue) { (& mise activate pwsh) | Out-String | Invoke-Expression }
if (Get-Command zoxide -ErrorAction SilentlyContinue) { (& zoxide init powershell) | Out-String | Invoke-Expression }
if (Get-Command tb -ErrorAction SilentlyContinue) { (& tb completion powershell) | Out-String | Invoke-Expression }

function glo { git log --oneline @args }
function repo { gh repo view --web @args }
function pr { tb pr @args }
function commit { tb commit @args }
function checkout { tb checkout @args }
function hot { tb hot @args }
