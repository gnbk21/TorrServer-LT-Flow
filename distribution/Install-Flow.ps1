param(
    [Parameter(Mandatory)][ValidateSet('stable', 'preview')][string]$Channel,
    [Parameter(Mandatory)][string]$InstallDirectory,
    [Parameter(Mandatory)][string]$StateDirectory,
    [ValidateRange(1, 65535)][int]$Port = 8090,
    [switch]$AsService,
    [switch]$HttpAuth,
    [string[]]$ListenAddress = @(),
    [switch]$RequireAttestation
)
. (Join-Path $PSScriptRoot 'FlowRelease.ps1')
$install = [IO.Path]::GetFullPath($InstallDirectory)
$state = [IO.Path]::GetFullPath($StateDirectory)
if ($install -eq $state) { throw 'Use separate executable and state directories.' }
foreach ($address in $ListenAddress) {
    $parsed = $null
    if (-not [Net.IPAddress]::TryParse($address,[ref]$parsed)) { throw 'Listen addresses must be IP literals.' }
}
if ($ListenAddress.Count -and '127.0.0.1' -notin $ListenAddress -and '0.0.0.0' -notin $ListenAddress) { throw 'Managed updates require a loopback listener; include 127.0.0.1 in -ListenAddress.' }
if (Test-Path -LiteralPath (Join-Path $install 'TorrServer-LT-windows-amd64.exe')) { throw 'Use Update-Flow.ps1 for an existing installation.' }
$manifest = Get-FlowRelease $Channel
[void](New-Item -ItemType Directory -Path $install -Force)
[void](New-Item -ItemType Directory -Path $state -Force)
if ($HttpAuth) {
    $accounts = Get-Content -LiteralPath (Join-Path $state 'accs.db') -Raw | ConvertFrom-Json
    $entries = @($accounts.PSObject.Properties)
    if (-not $entries.Count -or @($entries | Where-Object { -not $_.Name -or -not ($_.Value -is [string]) -or -not $_.Value }).Count) { throw '-HttpAuth requires an existing valid accs.db account map in the state directory.' }
}
$temporary = Join-Path $install ('download-' + [guid]::NewGuid().ToString('N') + '.exe')
try {
    Save-FlowBinary $manifest $temporary -RequireAttestation:$RequireAttestation
    $exe = Join-Path $install 'TorrServer-LT-windows-amd64.exe'
    Move-Item -LiteralPath $temporary -Destination $exe
    if ($AsService) {
        $arguments = @('--service','install','--path',$state,'--port',[string]$Port)
        if ($HttpAuth) { $arguments += '--httpauth' }
        foreach ($address in $ListenAddress) { $arguments += @('--ip',$address) }
        & $exe @arguments
        if ($LASTEXITCODE -ne 0) { throw 'Service installation failed. The downloaded executable is retained.' }
    }
    Write-FlowInstallRecord $install $state $Port $manifest ([bool]$AsService) ([bool]$HttpAuth) $ListenAddress
    Write-Output "Installed $($manifest.Version). State: $state. Port: $Port."
    if ($AsService) { Write-Output 'Start using --service start from an elevated terminal.' }
    else { $extra=''; if ($HttpAuth) { $extra=' --httpauth' }; foreach ($address in $ListenAddress) { $extra += " --ip $address" }; Write-Output "Start: & '$exe' --path '$state' --port $Port$extra" }
} finally { if (Test-Path -LiteralPath $temporary) { Remove-Item -LiteralPath $temporary } }
