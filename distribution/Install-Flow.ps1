param(
    [Parameter(Mandatory)][ValidateSet('stable', 'preview')][string]$Channel,
    [Parameter(Mandatory)][string]$InstallDirectory,
    [Parameter(Mandatory)][string]$StateDirectory,
    [ValidateRange(1, 65535)][int]$Port = 8090,
    [switch]$AsService,
    [switch]$RequireAttestation
)
. (Join-Path $PSScriptRoot 'FlowRelease.ps1')
$install = [IO.Path]::GetFullPath($InstallDirectory)
$state = [IO.Path]::GetFullPath($StateDirectory)
if ($install -eq $state) { throw 'Use separate executable and state directories.' }
if (Test-Path -LiteralPath (Join-Path $install 'TorrServer-LT-windows-amd64.exe')) { throw 'Use Update-Flow.ps1 for an existing installation.' }
$manifest = Get-FlowRelease $Channel
[void](New-Item -ItemType Directory -Path $install -Force)
[void](New-Item -ItemType Directory -Path $state -Force)
$temporary = Join-Path $install ('download-' + [guid]::NewGuid().ToString('N') + '.exe')
try {
    Save-FlowBinary $manifest $temporary -RequireAttestation:$RequireAttestation
    $exe = Join-Path $install 'TorrServer-LT-windows-amd64.exe'
    Move-Item -LiteralPath $temporary -Destination $exe
    if ($AsService) {
        & $exe --service install --path $state --port $Port
        if ($LASTEXITCODE -ne 0) { throw 'Service installation failed. The downloaded executable is retained.' }
    }
    Write-FlowInstallRecord $install $state $Port $manifest ([bool]$AsService)
    Write-Output "Installed $($manifest.Version). State: $state. Port: $Port."
    if ($AsService) { Write-Output 'Start using --service start from an elevated terminal.' }
    else { Write-Output "Start: & '$exe' --path '$state' --port $Port" }
} finally { if (Test-Path -LiteralPath $temporary) { Remove-Item -LiteralPath $temporary } }
