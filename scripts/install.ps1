& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'

    $repo = 'pmqueiroz/toolbelt'
    $baseUrl = if ($env:TOOLBELT_BASE_URL) { $env:TOOLBELT_BASE_URL } else { "https://github.com/$repo/releases/latest/download" }
    $installDir = if ($env:TOOLBELT_INSTALL_DIR) { $env:TOOLBELT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\toolbelt' }

    function Write-Step($message) { Write-Host '==> ' -ForegroundColor Magenta -NoNewline; Write-Host $message }

    $arch = switch ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "unsupported architecture $env:PROCESSOR_ARCHITECTURE" }
    }

    $archive = "tb_windows_$arch.zip"
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
    New-Item -ItemType Directory -Path $tmp | Out-Null

    try {
        Write-Step "downloading $archive"
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/$archive" -OutFile (Join-Path $tmp $archive)
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt')

        $line = Get-Content (Join-Path $tmp 'checksums.txt') | Where-Object { ($_ -split '\s+')[1] -eq $archive } | Select-Object -First 1
        if (-not $line) { throw "$archive is missing from checksums.txt" }
        $expected = ($line -split '\s+')[0]
        $actual = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp $archive)).Hash
        if ($actual -ne $expected) { throw "checksum mismatch for $archive" }

        Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath $tmp -Force
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        Copy-Item (Join-Path $tmp 'tb.exe') (Join-Path $installDir 'tb.exe') -Force
    }
    finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = if ($userPath) { $userPath -split ';' } else { @() }
    if ($entries -notcontains $installDir) {
        [Environment]::SetEnvironmentVariable('Path', (@($installDir) + $entries | Where-Object { $_ }) -join ';', 'User')
        Write-Step "added $installDir to your user PATH"
    }
    if (($env:Path -split ';') -notcontains $installDir) {
        $env:Path = "$installDir;$env:Path"
    }

    $tb = Join-Path $installDir 'tb.exe'
    Write-Step "installed $(& $tb version) to $installDir"

    Write-Step "run 'tb setup' to set up this machine"
}
