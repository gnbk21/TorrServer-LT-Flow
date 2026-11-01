# Shared release validation. Compatible with Windows PowerShell 5.1 and PowerShell 7.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Move-FlowExecutable {
    param([string]$LiteralPath, [string]$Destination)
    # Windows may retain a sharing lock briefly after process exit (for example
    # while scanning an image). Never overwrite, force termination or retry a
    # permission/path error. A persistent lock still enters normal rollback.
    $deadline = [DateTime]::UtcNow.AddSeconds(5)
    while ($true) {
        try { Move-Item -LiteralPath $LiteralPath -Destination $Destination -ErrorAction Stop; return }
        catch {
            $code = $_.Exception.HResult -band 0xffff
            if ($code -notin @(32,33) -or [DateTime]::UtcNow -ge $deadline) { throw }
            Start-Sleep -Milliseconds 100
        }
    }
}

function Get-FlowRelease {
    param([ValidateSet('stable', 'preview')][string]$Channel)
    $headers = @{ 'User-Agent' = 'TorrServer-Flow'; Accept = 'application/vnd.github+json' }
    # Invoke-RestMethod emits a JSON array as one pipeline object. Normalize
    # after assignment so selection receives each release on both PS 5.1/7.
    $endpoint = 'https://api.github.com/repos/gnbk21/TorrServer-LT-Flow/releases/latest'
    if ($Channel -eq 'preview') { $endpoint = 'https://api.github.com/repos/gnbk21/TorrServer-LT-Flow/releases?per_page=100' }
    $releaseResponse = Invoke-RestMethod -Uri $endpoint -Headers $headers -TimeoutSec 30
    $releases = @($releaseResponse)
    $tagPattern = '^MatriX\.(?:145|148)\.Flow-v[0-9]+\.[0-9]+\.[0-9]+$'
    if ($Channel -eq 'preview') {
        $tagPattern = '^MatriX\.(?:145|148)\.Flow-(v[0-9]+\.[0-9]+\.[0-9]+-(preview|alpha|beta|rc)\.[0-9]+|preview\.[0-9]+)$'
    }
    $eligible = @($releases | Where-Object { -not $_.draft -and ([bool]$_.prerelease -eq ($Channel -eq 'preview')) -and $_.tag_name -cmatch $tagPattern })
    $release = $null
    foreach ($candidate in $eligible) {
        if (-not $release -or (Compare-FlowVersions $candidate.tag_name $release.tag_name) -gt 0) { $release = $candidate }
    }
    if (-not $release) { throw "No published Flow $Channel release is available." }
    $asset = $release.assets | Where-Object { $_.name -eq 'release.json' } | Select-Object -First 1
    if (-not $asset) { throw 'This release has no verified update manifest. Download and install its package manually.' }
    $manifestUrl = 'https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/' + $release.tag_name + '/release.json'
    if ($asset.browser_download_url -ne $manifestUrl) { throw 'Unexpected manifest asset URL.' }
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
        $source = [string]$Manifest.commit
        $sourceRef = 'refs/tags/' + $Manifest.Version
        $workflow = 'ts_release.yml'
        if ($Manifest.PSObject.Properties.Name -contains 'packaging_workflow_commit') {
            $source = [string]$Manifest.packaging_workflow_commit
            if ($source -notmatch '^[a-f0-9]{40}$') { throw 'Invalid packaging source identity.' }
            $sourceRef = 'refs/heads/master'
            $workflow = 'recover-release.yml'
        }
        if ($Manifest.PSObject.Properties.Name -contains 'provenance') {
            $proof = $Manifest.provenance
            if ($proof.workflow -notin @('ts_release.yml','release-recovery.yml') -or $proof.source_commit -cne $source -or $proof.source_ref -cne $sourceRef) { throw 'Invalid release provenance policy.' }
            $workflow = [string]$proof.workflow
        }
        & $gh.Source attestation verify $Destination --repo gnbk21/TorrServer-LT-Flow --source-digest $source --source-ref $sourceRef --signer-workflow ('gnbk21/TorrServer-LT-Flow/.github/workflows/' + $workflow) --deny-self-hosted-runners
        if ($LASTEXITCODE -ne 0) { throw 'GitHub build provenance verification failed.' }
    }
    $version = (& $Destination --version | Out-String).Trim()
    $exactVersion = '(?<![A-Za-z0-9.-])'+[regex]::Escape([string]$Manifest.Version)+'(?![A-Za-z0-9.-])'
    if ($LASTEXITCODE -ne 0 -or $version -notmatch $exactVersion) { throw 'Executable version does not match the release.' }
}

function Compare-FlowVersions {
    param([string]$Left, [string]$Right)
    $values = @()
    foreach ($tag in @($Left,$Right)) {
        if ($tag -cmatch '^MatriX\.(?:145|148)\.Flow-v([0-9]+)\.([0-9]+)\.([0-9]+)(?:-(alpha|beta|preview|rc)\.([0-9]+))?$') {
            $rank = 4; $sequence = 0L
            if ($Matches[4]) { $rank = @('alpha','beta','preview','rc').IndexOf($Matches[4]); $sequence = [long]$Matches[5] }
            $values += ,@([long]$Matches[1],[long]$Matches[2],[long]$Matches[3],$rank,$sequence)
        } elseif ($tag -cmatch '^MatriX\.(?:145|148)\.Flow-preview\.([0-9]+)$') {
            $values += ,@(0L,0L,0L,2,[long]$Matches[1])
        } else { throw 'Cannot compare a non-release Flow version.' }
    }
    for ($i=0; $i -lt 5; $i++) {
        if ($values[0][$i] -lt $values[1][$i]) { return -1 }
        if ($values[0][$i] -gt $values[1][$i]) { return 1 }
    }
    return 0
}

# Version 1 inventory: only these application-owned names may participate in
# updater recovery. A shared state directory is never treated as owned wholesale.
function Test-FlowOwnedStateName([string]$Relative) {
    return $Relative -cin @('config.db','settings.json','viewed.json','accs.db','torrserver.db','flow-settings-good.json','flow-settings-pending.json','flow-storage-transaction.json','flow-preparation.json','flow-dht.bin','tg_langs.json','wip.txt','bip.txt','wip.txt.bak','bip.txt.bak','server.pem','server.key','ssl/uploaded.crt','ssl/uploaded.key') -or $Relative -cmatch '^ssl/pair-[A-Za-z0-9-]+/uploaded\.(crt|key)$'
}

function Resolve-FlowStateFile([string]$Directory, [string]$Relative) {
    if (-not (Test-FlowOwnedStateName $Relative)) { throw 'Unrecognized Flow state filename.' }
    $root = [IO.Path]::GetFullPath($Directory)
    if ($root.Length -gt [IO.Path]::GetPathRoot($root).Length) { $root = $root.TrimEnd('\','/') }
    $ancestor=$root
    while ($ancestor) {
        if (Test-Path -LiteralPath $ancestor) {
            $entry=Get-Item -LiteralPath $ancestor -Force
            if (($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Flow state ancestors must not contain reparse points.' }
        }
        $parent=Split-Path $ancestor -Parent
        if ($parent -eq $ancestor) { break }
        $ancestor=$parent
    }
    $current = $root
    foreach ($part in @('') + @($Relative.Split('/'))) {
        if ($part) { $current = Join-Path $current $part }
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or (($item.PSObject.Properties.Name -contains 'LinkType') -and $item.LinkType -eq 'HardLink')) { throw 'Flow update state paths must not contain reparse points.' }
        }
    }
    $path = [IO.Path]::GetFullPath($current)
    if (-not $path.StartsWith($root.TrimEnd('\','/')+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)) { throw 'State path escaped its directory.' }
    return $path
}

function Get-FlowOwnedState([string]$Directory) {
    $names = @('config.db','settings.json','viewed.json','accs.db','torrserver.db','flow-settings-good.json','flow-settings-pending.json','flow-storage-transaction.json','flow-preparation.json','flow-dht.bin','tg_langs.json','wip.txt','bip.txt','wip.txt.bak','bip.txt.bak','server.pem','server.key','ssl/uploaded.crt','ssl/uploaded.key')
    # Validate ssl before listing; never traverse a directory junction.
    [void](Resolve-FlowStateFile $Directory 'ssl/uploaded.crt')
    $ssl = Join-Path $Directory 'ssl'
    if (Test-Path -LiteralPath $ssl -PathType Container) {
        foreach ($pair in @(Get-ChildItem -LiteralPath $ssl -Directory -Force | Where-Object { $_.Name -cmatch '^pair-[A-Za-z0-9-]+$' })) {
            $names += 'ssl/'+$pair.Name+'/uploaded.crt', 'ssl/'+$pair.Name+'/uploaded.key'
        }
    }
    if ($names.Count -gt 512) { throw 'Flow state inventory exceeds its bound.' }
    foreach ($name in $names) {
        $path = Resolve-FlowStateFile $Directory $name
        if (Test-Path -LiteralPath $path) {
            if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw 'A Flow state filename is not a regular file.' }
            [pscustomobject]@{path=$name;sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()}
        }
    }
}

function Save-FlowStateSnapshot([string]$State, [string]$Backup) {
    [void](Resolve-FlowStateFile $Backup 'settings.json')
    $inventoryPath = Join-Path $Backup 'inventory.json'
    if (Test-Path -LiteralPath $inventoryPath) { throw 'Recovery inventory must be new.' }
    $files = @(Get-FlowOwnedState $State)
    foreach ($file in $files) {
        $destination = Resolve-FlowStateFile $Backup $file.path
        [void](New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force)
        Copy-Item -LiteralPath (Resolve-FlowStateFile $State $file.path) -Destination $destination
        if ((Get-FileHash -LiteralPath $destination).Hash.ToLowerInvariant() -cne $file.sha256) { throw 'State changed while capturing the update backup.' }
    }
    @{schema_version=1;files=@($files)} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $inventoryPath -Encoding UTF8
}

function Restore-FlowStateSnapshot([string]$State, [string]$Backup) {
    [void](Resolve-FlowStateFile $Backup 'settings.json')
    $inventoryPath=Join-Path $Backup 'inventory.json'
    $inventoryFile=Get-Item -LiteralPath $inventoryPath -Force
    if (($inventoryFile.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or $inventoryFile.Length -gt 1MB -or $inventoryFile.LinkType -eq 'HardLink') { throw 'Invalid recovery inventory file.' }
    $inventory = Get-Content -LiteralPath $inventoryPath -Raw | ConvertFrom-Json
    if ($inventory.schema_version -ne 1 -or @($inventory.files).Count -gt 512) { throw 'Invalid recovery inventory.' }
    $names = @()
    foreach ($file in @($inventory.files)) {
        if ($file.path -in $names -or $file.sha256 -cnotmatch '^[a-f0-9]{64}$') { throw 'Invalid recovery entry.' }
        $names += $file.path
        $path = Resolve-FlowStateFile $Backup $file.path
        [void](Resolve-FlowStateFile $State $file.path)
        if ((Get-FileHash -LiteralPath $path).Hash.ToLowerInvariant() -cne $file.sha256) { throw 'Recovery file checksum mismatch.' }
    }
    # Quarantine only newly created, recognized Flow files. Unrelated files,
    # even those created while updating, remain entirely outside recovery.
    foreach ($file in @(Get-FlowOwnedState $State)) {
        if ($file.path -cin $names) { continue }
        $destination = Resolve-FlowStateFile (Join-Path $Backup 'failed-state') $file.path
        [void](New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force)
        Move-Item -LiteralPath (Resolve-FlowStateFile $State $file.path) -Destination $destination
    }
    foreach ($file in @($inventory.files)) {
        $destination = Resolve-FlowStateFile $State $file.path
        [void](New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force)
        Copy-Item -LiteralPath (Resolve-FlowStateFile $Backup $file.path) -Destination $destination -Force
    }
}

function Write-FlowInstallRecord {
    param([string]$Directory, [string]$StateDirectory, [int]$Port, $Manifest, [bool]$Service, [bool]$HttpAuth = $false, [string[]]$ListenAddress = @())
    $record = @{ schema_version = 1; version = $Manifest.Version; channel = $Manifest.channel; commit = $Manifest.commit; state_directory = $StateDirectory; port = $Port; service = $Service; http_auth = $HttpAuth; listen_addresses = @($ListenAddress) }
    $record | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Directory 'flow-install.json') -Encoding UTF8
}
