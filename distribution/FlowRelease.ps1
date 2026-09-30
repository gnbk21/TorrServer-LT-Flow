# Shared release validation. Compatible with Windows PowerShell 5.1 and PowerShell 7.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-FlowRelease {
    param([ValidateSet('stable', 'preview')][string]$Channel)
    $headers = @{ 'User-Agent' = 'TorrServer-Flow'; Accept = 'application/vnd.github+json' }
    $releases = @(Invoke-RestMethod -Uri 'https://api.github.com/repos/gnbk21/TorrServer-LT-Flow/releases?per_page=100' -Headers $headers -TimeoutSec 30)
    $release = $releases | Where-Object { -not $_.draft -and ([bool]$_.prerelease -eq ($Channel -eq 'preview')) -and $_.tag_name -match '^MatriX\.145\.Flow-' } | Select-Object -First 1
    if (-not $release) { throw "No published Flow $Channel release is available." }
    $asset = $release.assets | Where-Object { $_.name -eq 'release.json' } | Select-Object -First 1
    if (-not $asset) { throw 'This release has no verified update manifest. Download and install its package manually.' }
    $manifest = Invoke-RestMethod -Uri $asset.browser_download_url -Headers $headers -TimeoutSec 30
    if ($manifest.schema_version -ne 1 -or $manifest.repository -ne 'gnbk21/TorrServer-LT-Flow' -or $manifest.channel -ne $Channel -or $manifest.Version -ne $release.tag_name -or $manifest.commit -notmatch '^[a-f0-9]{40}$') {
        throw 'Invalid Flow release identity or channel.'
    }
    return $manifest
}

function Save-FlowBinary {
    param($Manifest, [string]$Destination, [switch]$RequireAttestation)
    $name = 'TorrServer-LT-windows-amd64.exe'
    $file = $Manifest.files.$name
    $expectedUrl = 'https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/' + $Manifest.Version + '/' + $name
    if (-not $file -or $file.url -ne $expectedUrl -or $file.sha256 -notmatch '^[a-f0-9]{64}$' -or [long]$file.size -le 0 -or [long]$file.size -gt 256MB) {
        throw 'Invalid executable manifest entry.'
    }
    Invoke-WebRequest -UseBasicParsing -Uri $expectedUrl -OutFile $Destination -TimeoutSec 120
    if ((Get-Item -LiteralPath $Destination).Length -ne [long]$file.size -or (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash.ToLowerInvariant() -ne $file.sha256) {
        Remove-Item -LiteralPath $Destination
        throw 'Executable integrity check failed.'
    }
    if ($RequireAttestation) {
        $gh = Get-Command gh -ErrorAction Stop
        & $gh.Source attestation verify $Destination --repo gnbk21/TorrServer-LT-Flow
        if ($LASTEXITCODE -ne 0) { throw 'GitHub build provenance verification failed.' }
    }
    $version = (& $Destination --version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $version -notlike ('*' + $Manifest.Version + '*')) { throw 'Executable version does not match the release.' }
}

function Write-FlowInstallRecord {
    param([string]$Directory, [string]$StateDirectory, [int]$Port, $Manifest, [bool]$Service)
    $record = @{ schema_version = 1; version = $Manifest.Version; channel = $Manifest.channel; commit = $Manifest.commit; state_directory = $StateDirectory; port = $Port; service = $Service }
    $record | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Directory 'flow-install.json') -Encoding UTF8
}
