# Usage: irm https://raw.githubusercontent.com/takashi145/chess-replay/main/install.ps1 | iex

# Wrapped in a script block so variables and preference changes don't leak into the caller's session when run via iex.
# It also means a partially downloaded script fails to parse instead of running halfway.
& {
    $ErrorActionPreference = 'Stop'
    # The progress bar makes Invoke-WebRequest drastically slower on Windows PowerShell 5.1.
    $ProgressPreference = 'SilentlyContinue'

    # Asset names must match the ones produced by .github/workflows/release.yml.
    $baseUrl = 'https://github.com/takashi145/chess-replay/releases/latest/download'
    $asset = 'chess-replay-win-x64.zip'
    $installDir = Join-Path $env:LOCALAPPDATA 'Programs\chess-replay'
    $exePath = Join-Path $installDir 'chess-replay.exe'

    # Windows PowerShell 5.1 may not enable TLS 1.2 by default, which GitHub requires.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null

    # Download to temp files first so a failed or tampered download never replaces the installed exe.
    $zipPath = [IO.Path]::GetTempFileName()
    $sumsPath = [IO.Path]::GetTempFileName()
    $extractDir = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
    try {
        Write-Host "Downloading chess-replay..."
        Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $zipPath -UseBasicParsing
        Invoke-WebRequest -Uri "$baseUrl/SHA256SUMS" -OutFile $sumsPath -UseBasicParsing

        $pattern = '^([0-9a-fA-F]{64})\s+\*?' + [regex]::Escape($asset) + '$'
        $match = Get-Content $sumsPath | Select-String -Pattern $pattern | Select-Object -First 1
        if (-not $match) { throw "Checksum for $asset not found in SHA256SUMS." }
        $expected = $match.Matches[0].Groups[1].Value
        $actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash
        if ($actual -ne $expected) { throw "Checksum mismatch for $asset. Aborting installation." }

        # Expand-Archive requires a .zip extension on Windows PowerShell 5.1, so go through .NET instead.
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)
        # The binary bundles third-party code, so keep its license files next to it.
        Copy-Item -Force -Path (Join-Path $extractDir 'LICENSE'), (Join-Path $extractDir 'THIRD-PARTY-NOTICES.md') -Destination $installDir
        Move-Item -Force -Path (Join-Path $extractDir 'chess-replay.exe') -Destination $exePath
    }
    finally {
        Remove-Item -Force -Recurse -ErrorAction SilentlyContinue -Path $zipPath, $sumsPath, $extractDir
    }

    # Read and write the user PATH through the registry without expanding it.
    # [Environment]::Get/SetEnvironmentVariable would expand entries like %USERPROFILE%\... and
    # save them back as fixed paths, and turn the value from REG_EXPAND_SZ into REG_SZ.
    $envKey = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    try {
        $rawPath = [string]$envKey.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $entries = $rawPath -split ';' | Where-Object { $_ } | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') }

        if ($entries -notcontains $installDir.TrimEnd('\')) {
            $kind = if ($envKey.GetValueNames() -contains 'Path') { $envKey.GetValueKind('Path') } else { [Microsoft.Win32.RegistryValueKind]::ExpandString }
            # Append only; existing entries are kept exactly as they are.
            $newPath = if (-not $rawPath) { $installDir } elseif ($rawPath.EndsWith(';')) { "$rawPath$installDir" } else { "$rawPath;$installDir" }
            $envKey.SetValue('Path', $newPath, $kind)
            $env:Path = "$env:Path;$installDir"

            # Writing the registry directly doesn't notify other apps (e.g. Explorer) the way
            # SetEnvironmentVariable does, so broadcast WM_SETTINGCHANGE ourselves. Failure here is harmless.
            try {
                if (-not ('ChessReplayInstall.NativeMethods' -as [type])) {
                    Add-Type -Namespace ChessReplayInstall -Name NativeMethods -MemberDefinition '
                        [DllImport("user32.dll", CharSet = CharSet.Unicode)]
                        public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint msg, UIntPtr wParam, string lParam, uint flags, uint timeout, out UIntPtr result);'
                }
                $result = [UIntPtr]::Zero
                [void][ChessReplayInstall.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result)
            }
            catch { }

            Write-Host "Added $installDir to your PATH. Restart other open terminals to pick it up."
        }
    }
    finally {
        $envKey.Close()
    }

    Write-Host "Installed chess-replay to $exePath"
    Write-Host "Try: chess-replay <username>"
}
