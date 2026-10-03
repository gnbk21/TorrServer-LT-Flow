param(
    [Parameter(Mandatory)][string]$InstallDirectory,
    [Parameter(Mandatory)][ValidateSet('stable', 'preview')][string]$Channel,
    [System.Management.Automation.PSCredential]$Credential,
    [switch]$RequireAttestation
)
. (Join-Path $PSScriptRoot 'FlowRelease.ps1')
$install = [IO.Path]::GetFullPath($InstallDirectory)
$recordPath = Join-Path $install 'flow-install.json'
$record = Get-Content -LiteralPath $recordPath -Raw | ConvertFrom-Json
if ($record.schema_version -ne 1) { throw 'Unsupported installation record.' }
$httpAuth = ($record.PSObject.Properties.Name -contains 'http_auth') -and [bool]$record.http_auth
$listenAddresses = @()
if ($record.PSObject.Properties.Name -contains 'listen_addresses') { $listenAddresses = @($record.listen_addresses) }
foreach ($address in $listenAddresses) {
    $parsed = $null
    if (-not [Net.IPAddress]::TryParse([string]$address,[ref]$parsed)) { throw 'Invalid recorded listener address.' }
}
if ($httpAuth -and -not $Credential) { throw 'This installation requires -Credential for authenticated health and maintenance checks.' }
$exe = Join-Path $install 'TorrServer-LT-windows-amd64.exe'
$manifest = Get-FlowRelease $Channel
if ($manifest.Version -eq $record.version) { Write-Output 'This release is already installed.'; return }
$request = @{ TimeoutSec = 5; ErrorAction = 'Stop' }
if ($Credential) {
    # PowerShell 7 forbids -Credential on HTTP. Send Basic only to the fixed
    # loopback endpoint, preserving compatibility with PowerShell 5.1.
    $authentication = $Credential.GetNetworkCredential()
    $request.Headers = @{ Authorization = 'Basic '+[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($Credential.UserName+':'+$authentication.Password)) }
    $authentication = $null
}
$baseUri = 'http://127.0.0.1:' + [int]$record.port
$maintenanceToken = ''
function Set-FlowMaintenance([bool]$Enabled) {
    Invoke-RestMethod @request -Uri "$baseUri/flow/maintenance" -Method Post -ContentType 'application/json' -Body (@{enabled=$Enabled;token=$maintenanceToken} | ConvertTo-Json -Compress)
}
function Start-InstalledFlow {
    if ($record.service) {
        & $exe --service start
        if ($LASTEXITCODE -ne 0) { throw 'Service startup failed.' }
    } else {
        # Arguments are quoted for Windows CreateProcess, never evaluated as shell code.
        if ([string]$record.state_directory -match '["\r\n]') { throw 'Invalid state path.' }
        $arguments = '--path "' + ([string]$record.state_directory).TrimEnd('\') + '" --port ' + [int]$record.port
        if ($httpAuth) { $arguments += ' --httpauth' }
        foreach ($address in $listenAddresses) { $arguments += ' --ip '+$address }
        Start-Process -FilePath $exe -ArgumentList $arguments -WorkingDirectory $install -WindowStyle Hidden | Out-Null
    }
}
function Wait-FlowHealth([string]$ExpectedVersion) {
    $deadline = [DateTime]::UtcNow.AddSeconds(30)
    do {
        try {
            $version = Invoke-RestMethod @request -Uri "$baseUri/echo"
            $status = Invoke-RestMethod @request -Uri "$baseUri/flow/tray"
            if ($version -eq $ExpectedVersion -and $status.server_state -in @('RUNNING','PAUSED')) { return }
        } catch { }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw 'Updated server did not become healthy within 30 seconds.'
}
$temporary = Join-Path $install ('download-' + [guid]::NewGuid().ToString('N') + '.exe')
$previous = Join-Path $install ('previous-' + [guid]::NewGuid().ToString('N') + '.exe')
$quiesced = $false; $swapped = $false; $stopped = $false; $backup = $null
try {
    Save-FlowBinary $manifest $temporary -RequireAttestation:$RequireAttestation
    if ($record.service) {
        # Replacement files do not inherit the previous executable's explicit
        # virtual-account grant. Preserve read/execute access before the swap.
        & icacls.exe $temporary /grant 'NT SERVICE\TorrServer-Flow:RX' | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Cannot grant the restricted service access to the candidate executable.' }
    }
    # No replacement until the server atomically rejects new playback and confirms idle.
    $maintenance = Set-FlowMaintenance $true
    if (-not $maintenance.enabled -or $maintenance.active_requests -ne 0) { throw 'Server is not idle.' }
    $quiesced = $true
    if ($maintenance.PSObject.Properties.Name -notcontains 'token' -or -not $maintenance.token) { throw 'This server lacks maintenance leases; release maintenance and use a manual upgrade.' }
    $maintenanceToken = $maintenance.token
    $processes = @(Get-Process -Name 'TorrServer-LT-windows-amd64' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe })
    if (-not $record.service -and $processes.Count -ne 1) { throw 'Expected exactly one managed server process.' }
    if ($record.service) {
        & $exe --service stop
        if ($LASTEXITCODE -ne 0) { throw 'Service shutdown failed.' }
    } else {
        try { Invoke-RestMethod @request -Uri "$baseUri/shutdown" | Out-Null } catch { }
        if (-not $processes[0].WaitForExit(30000)) { throw 'Server did not shut down; no replacement was made.' }
    }
    $stopped = $true
    # State remains in place. Keep a recovery copy of configuration after a clean close.
    $backup = Join-Path ([string]$record.state_directory) ('upgrade-backups/' + [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss') + '-' + [guid]::NewGuid().ToString('N'))
    [void](New-Item -ItemType Directory -Path $backup -Force)
    # Protect the directory before copying account/configuration files into it.
    $owner = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($sid in @($owner, 'S-1-5-18', 'S-1-5-32-544')) {
        $rule = New-Object Security.AccessControl.FileSystemAccessRule -ArgumentList ([Security.Principal.SecurityIdentifier]$sid), 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow'
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $backup -AclObject $acl
    Get-ChildItem -LiteralPath ([string]$record.state_directory) -File | Where-Object { $_.Extension -in @('.db','.json','.txt') } | Copy-Item -Destination $backup
    Move-Item -LiteralPath $exe -Destination $previous
    try { Move-Item -LiteralPath $temporary -Destination $exe } catch { Move-Item -LiteralPath $previous -Destination $exe; throw }
    $swapped = $true
    Start-InstalledFlow
    Wait-FlowHealth $manifest.Version
    Write-FlowInstallRecord $install ([string]$record.state_directory) ([int]$record.port) $manifest ([bool]$record.service) $httpAuth $listenAddresses
    Write-Output "Updated to $($manifest.Version). Previous executable: $previous. Configuration backup: $backup."
} catch {
    $failure = $_
    if ($swapped) {
        if ($record.service) {
            & $exe --service stop | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'Rollback blocked: service could not be stopped. Previous executable retained.' }
        }
        else {
            try { Invoke-RestMethod @request -Uri "$baseUri/shutdown" | Out-Null } catch { }
            $new = @(Get-Process -Name 'TorrServer-LT-windows-amd64' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe })
            foreach ($process in $new) { if (-not $process.WaitForExit(30000)) { throw 'Rollback blocked: updated server is still running. Previous executable retained.' } }
        }
        $failed = Join-Path $install ('failed-' + [guid]::NewGuid().ToString('N') + '.exe')
        Move-Item -LiteralPath $exe -Destination $failed
        Move-Item -LiteralPath $previous -Destination $exe
        # Undo migrations of files captured after the old server closed cleanly.
        $originalNames = @(Get-ChildItem -LiteralPath $backup -File | ForEach-Object { $_.Name })
        $createdFiles = @(Get-ChildItem -LiteralPath ([string]$record.state_directory) -File | Where-Object { $_.Extension -in @('.db','.json','.txt') -and $_.Name -notin $originalNames })
        if ($createdFiles.Count) {
            # Preserve new migration files privately, but remove them from the
            # old server's state namespace before restoring its snapshot.
            $failedState = Join-Path $backup 'failed-state'
            [void](New-Item -ItemType Directory -Path $failedState)
            foreach ($file in $createdFiles) { Move-Item -LiteralPath $file.FullName -Destination (Join-Path $failedState $file.Name) }
        }
        Get-ChildItem -LiteralPath $backup -File | Copy-Item -Destination ([string]$record.state_directory) -Force
        Start-InstalledFlow
        Wait-FlowHealth ([string]$record.version)
        Write-Warning 'Previous executable restored and health checked. State was preserved.'
    } elseif ($stopped) {
        Start-InstalledFlow
        Wait-FlowHealth ([string]$record.version)
        Write-Warning 'The previous executable was restarted after an update preparation failure.'
    } elseif ($quiesced) { try { Set-FlowMaintenance $false | Out-Null } catch { } }
    throw $failure
} finally { if (Test-Path -LiteralPath $temporary) { Remove-Item -LiteralPath $temporary } }
