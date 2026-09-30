param(
    [Parameter(Mandatory)][string]$Baseline,
    [Parameter(Mandatory)][string]$Candidate,
    [Parameter(Mandatory)][string]$OutputDirectory,
    [string]$Commit = '',
    [switch]$AsService
)
$ErrorActionPreference = 'Stop'
if ($AsService -and ($env:GITHUB_ACTIONS -ne 'true' -or (Get-Service -Name 'TorrServer-Flow' -ErrorAction SilentlyContinue))) { throw 'Service update verification requires a disposable CI runner with no existing Flow service.' }
$root = Split-Path $PSScriptRoot -Parent
$baselinePath = (Resolve-Path -LiteralPath $Baseline).Path
$candidatePath = (Resolve-Path -LiteralPath $Candidate).Path
$output = [IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $output) { throw 'Test output must be new.' }
[void](New-Item -ItemType Directory -Path $output)
$install = Join-Path $output 'bin'
$state = Join-Path $output 'state'
[void](New-Item -ItemType Directory -Path $state)
$exe = Join-Path $install 'TorrServer-LT-windows-amd64.exe'
$reservation = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0)
$reservation.Start(); $port=$reservation.LocalEndpoint.Port; $reservation.Stop()
$baseUri = 'http://127.0.0.1:'+$port
$credential = [PSCredential]::new('fixture',(ConvertTo-SecureString 'update-fixture-only' -AsPlainText -Force))
$headers = @{Authorization='Basic '+[Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes('fixture:update-fixture-only'))}
@{fixture='update-fixture-only'} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $state 'accs.db') -Encoding ASCII
@{BitTorr=@{DisableDHT=$true;DisablePEX=$true;DisableUPNP=$true;EnableBonjour=$false;EnableDLNA=$false;RetrackersMode=0;DefaultTrackers='';JacRedKey='retained-private-fixture'}} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $state 'settings.json') -Encoding ASCII

function Manifest([string]$Path) {
    $versionText = (& $Path --version | Out-String)
    if ($LASTEXITCODE -ne 0 -or $versionText -notmatch '(MatriX\.145\.Flow-[a-zA-Z0-9.-]+)') { throw 'Use two native Flow binaries with different identities.' }
    $version=$Matches[1]; $identity=$Commit
    if ($version -match '^MatriX\.145\.Flow-dev-([a-f0-9]{40})') { $identity=$Matches[1] }
    if ($identity -notmatch '^[a-f0-9]{40}$') { throw 'Supply the verified CI commit for release binaries.' }
    [pscustomobject]@{schema_version=1;repository='gnbk21/TorrServer-LT-Flow';channel='preview';Version=$version;commit=$identity;files=[pscustomobject]@{'TorrServer-LT-windows-amd64.exe'=[pscustomobject]@{url="https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/$version/TorrServer-LT-windows-amd64.exe";sha256=(Get-FileHash -LiteralPath $Path).Hash.ToLowerInvariant();size=(Get-Item -LiteralPath $Path).Length}}}
}
$oldManifest = Manifest $baselinePath
$newManifest = Manifest $candidatePath
if ($oldManifest.Version -eq $newManifest.Version) { throw 'Test binaries must have different build identities.' }
$previewTag = '^MatriX\.145\.Flow-(v[0-9]+\.[0-9]+\.[0-9]+-(preview|alpha|beta|rc)\.[0-9]+|preview\.[0-9]+)$'
if ($oldManifest.Version -cnotmatch $previewTag -or $newManifest.Version -cnotmatch $previewTag) { throw 'Update transport fixtures require valid prerelease identities; development artifacts are deliberately not releases.' }
$global:flowUpdateTestfixtureManifest=$oldManifest
$global:flowUpdateTestfixtureBinary=$baselinePath
$global:flowUpdateTestcorrupt=$false
$global:flowUpdateTestfailNextStart=$false
$global:flowUpdateTeststartArguments=@()

# Replace only the remote release transport. Validation, binary execution,
# authenticated native HTTP, file swaps, protected backups and rollback use
# the production scripts unchanged. This is not a live GitHub release test.
function Invoke-RestMethod {
    [CmdletBinding()]param([string]$Uri,[hashtable]$Headers,[int]$TimeoutSec,[string]$Method,[string]$ContentType,$Body)
    if ($Uri -eq 'https://api.github.com/repos/gnbk21/TorrServer-LT-Flow/releases?per_page=100') {
        return @([pscustomobject]@{draft=$false;prerelease=$true;tag_name=$global:flowUpdateTestfixtureManifest.Version;assets=@([pscustomobject]@{name='release.json';browser_download_url="https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/$($global:flowUpdateTestfixtureManifest.Version)/release.json"})})
    }
    if ($Uri -eq "https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/$($global:flowUpdateTestfixtureManifest.Version)/release.json") { return $global:flowUpdateTestfixtureManifest }
    Microsoft.PowerShell.Utility\Invoke-RestMethod @PSBoundParameters
}
function Invoke-WebRequest {
    [CmdletBinding()]param([switch]$UseBasicParsing,[string]$Uri,[string]$OutFile,[int]$TimeoutSec)
    if ($Uri -ne $global:flowUpdateTestfixtureManifest.files.'TorrServer-LT-windows-amd64.exe'.url) { throw 'Unexpected download request.' }
    Copy-Item -LiteralPath $global:flowUpdateTestfixtureBinary -Destination $OutFile
    if ($global:flowUpdateTestcorrupt) { $stream=[IO.File]::OpenWrite($OutFile); try {$stream.Position=$stream.Length; $stream.WriteByte(0)} finally {$stream.Dispose()} }
}
function Start-Process {
    [CmdletBinding()]param([string]$FilePath,[string]$ArgumentList,[string]$WorkingDirectory,[string]$WindowStyle)
    $global:flowUpdateTeststartArguments += $ArgumentList
    if ($global:flowUpdateTestfailNextStart) {
        $global:flowUpdateTestfailNextStart=$false
        # Simulate a candidate migration after the pre-update snapshot. The
        # rollback must restore data as well as the previous executable.
        [IO.File]::WriteAllBytes((Join-Path $state 'config.db'),[Text.Encoding]::ASCII.GetBytes('candidate-migration-fixture'))
        [IO.File]::WriteAllText((Join-Path $state 'candidate-only.json'),'{"migration":"fixture"}')
        # Inject a CLI parsing failure into the actual staged executable.
        $PSBoundParameters.ArgumentList += ' --port invalid'
    }
    Microsoft.PowerShell.Management\Start-Process @PSBoundParameters
}
function Move-Item {
    [CmdletBinding()]param([string]$LiteralPath,[string]$Destination,[switch]$Force)
    Microsoft.PowerShell.Management\Move-Item @PSBoundParameters
    if ($AsService -and $global:flowUpdateTestfailNextStart -and $Destination -eq $exe -and [IO.Path]::GetFileName($LiteralPath).StartsWith('download-')) {
        $global:flowUpdateTestfailNextStart=$false
        [IO.File]::WriteAllBytes((Join-Path $state 'config.db'),[Text.Encoding]::ASCII.GetBytes('candidate-migration-fixture'))
        [IO.File]::WriteAllText((Join-Path $state 'candidate-only.json'),'{"migration":"fixture"}')
    }
}
function Health([string]$Version) {
    $deadline=[DateTime]::UtcNow.AddSeconds(30)
    do {
        try {
            $actual=Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/echo" -Headers $headers -TimeoutSec 2
            $tray=Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/flow/tray" -Headers $headers -TimeoutSec 2
            if ($actual -eq $Version -and $tray.server_state -eq 'RUNNING') { return }
        } catch { }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)
    throw 'Owned test server did not become ready.'
}
function CheckState {
    $settings=Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/settings" -Method Post -Body '{"action":"get"}' -ContentType 'application/json' -Headers $headers
    if ($settings.JacRedKey -ne 'retained-private-fixture') { throw 'Private state was changed.' }
    $record=Get-Content -LiteralPath (Join-Path $install 'flow-install.json') -Raw | ConvertFrom-Json
    if (-not $record.http_auth -or @($record.listen_addresses).Count -ne 1 -or $record.listen_addresses[0] -ne '127.0.0.1') { throw 'Authentication/listener configuration was lost.' }
}
try {
    & (Join-Path $root 'distribution/Install-Flow.ps1') -Channel preview -InstallDirectory $install -StateDirectory $state -Port $port -HttpAuth -ListenAddress '127.0.0.1' -AsService:$AsService
    $arguments='--path "'+$state+'" --port '+$port+' --ip 127.0.0.1 --httpauth'
    if ($AsService) {
        & $exe --service start
        if ($LASTEXITCODE -ne 0) { throw 'Owned test service did not start.' }
    } else { Microsoft.PowerShell.Management\Start-Process -FilePath $exe -ArgumentList $arguments -WindowStyle Hidden }
    Health $oldManifest.Version
    CheckState
    if ($AsService) {
        $service=Get-CimInstance Win32_Service -Filter "Name='TorrServer-Flow'"
        if ($service.StartName -ne 'NT SERVICE\TorrServer-Flow' -or $service.PathName -notmatch '--httpauth' -or $service.PathName -notmatch '--ip 127\.0\.0\.1') { throw 'Service update lost its identity or listener/authentication flags.' }
    }
    $global:flowUpdateTestfixtureManifest=$newManifest; $global:flowUpdateTestfixtureBinary=$candidatePath
    $lease=Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/flow/maintenance" -Method Post -Body '{"enabled":true}' -ContentType 'application/json' -Headers $headers
    $busyRejected=$false
    try { & (Join-Path $root 'distribution/Update-Flow.ps1') -InstallDirectory $install -Channel preview -Credential $credential } catch { $busyRejected=$true }
    if (-not $busyRejected) { throw 'Updater ignored another maintenance owner.' }
    Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/flow/maintenance" -Method Post -Body (@{enabled=$false;token=$lease.token}|ConvertTo-Json -Compress) -ContentType 'application/json' -Headers $headers | Out-Null
    Health $oldManifest.Version
    $global:flowUpdateTestcorrupt=$true
    $rejected=$false
    try { & (Join-Path $root 'distribution/Update-Flow.ps1') -InstallDirectory $install -Channel preview -Credential $credential } catch { $rejected=$_.Exception.Message -match 'integrity' }
    if (-not $rejected) { throw 'Corrupted download was accepted.' }
    Health $oldManifest.Version
    $global:flowUpdateTestcorrupt=$false
    & (Join-Path $root 'distribution/Update-Flow.ps1') -InstallDirectory $install -Channel preview -Credential $credential
    Health $newManifest.Version
    CheckState
    $global:flowUpdateTestfixtureManifest=$oldManifest; $global:flowUpdateTestfixtureBinary=$baselinePath; $global:flowUpdateTestfailNextStart=$true
    $rolledBack=$false
    try { & (Join-Path $root 'distribution/Update-Flow.ps1') -InstallDirectory $install -Channel preview -Credential $credential } catch { $rolledBack=$true; $rollbackError=$_.Exception.Message }
    if (-not $rolledBack) { throw 'Injected startup failure did not trigger rollback.' }
    if ($global:flowUpdateTestfailNextStart) { throw 'Candidate migration failure was not exercised.' }
    Health $newManifest.Version
    CheckState
    if ($AsService) {
        $service=Get-CimInstance Win32_Service -Filter "Name='TorrServer-Flow'"
        if ($service.StartName -ne 'NT SERVICE\TorrServer-Flow' -or $service.PathName -notmatch '--httpauth' -or $service.PathName -notmatch '--ip 127\.0\.0\.1') { throw 'Rollback changed the service account or listener/authentication flags.' }
    }
    if (Test-Path -LiteralPath (Join-Path $state 'candidate-only.json')) { throw 'Failed candidate migration file remained in active state.' }
    $retainedMigration = @(Get-ChildItem -LiteralPath (Join-Path $state 'upgrade-backups') -Recurse -File -Filter 'candidate-only.json')
    if ($retainedMigration.Count -ne 1 -or $retainedMigration[0].Directory.Name -ne 'failed-state') { throw 'Failed migration data was not preserved privately.' }
    foreach ($arguments in $global:flowUpdateTeststartArguments) {
        if ($arguments -notmatch '--httpauth' -or $arguments -notmatch '--ip 127\.0\.0\.1') { throw 'Restart changed authentication or listener flags.' }
    }
    $backups=@(Get-ChildItem -LiteralPath (Join-Path $state 'upgrade-backups') -Directory)
    if ($backups.Count -ne 2) { throw 'Protected update backups missing.' }
    foreach ($backup in $backups) {
        $acl=Get-Acl -LiteralPath $backup.FullName
        if (-not $acl.AreAccessRulesProtected) { throw 'Recovery directory inherits unsafe access.' }
        if ($acl.Access | Where-Object {$_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value -in @('S-1-1-0','S-1-5-11','S-1-5-32-545')}) { throw 'Recovery backup is readable by general users.' }
    }
    @{passed=$true;native_executables=$true;service=$AsService.IsPresent;release_transport='controlled fixture';maintenance_owner_rejection=$true;integrity_rejection=$true;authenticated_update=$true;startup_failure_rollback=$true;configuration_mutation_rollback=$true;failed_migration_quarantined=$true;state_preserved=$true;listener_preserved=$true;protected_backups=$true;rollback_error=$rollbackError} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $output 'report.json')
} finally {
    try { Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri "$baseUri/shutdown" -Headers $headers -TimeoutSec 2 | Out-Null } catch { }
    $owned=@(Get-Process -Name 'TorrServer-LT-windows-amd64' -ErrorAction SilentlyContinue | Where-Object {$_.Path -eq $exe})
    foreach ($process in $owned) { if (-not $process.WaitForExit(30000)) { $process.Kill(); $process.WaitForExit() } }
    if ($AsService -and (Get-Service -Name 'TorrServer-Flow' -ErrorAction SilentlyContinue)) {
        & $exe --service uninstall
        if ($LASTEXITCODE -ne 0) { throw 'Owned test service cleanup failed.' }
    }
}
