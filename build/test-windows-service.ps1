param([Parameter(Mandatory)][string]$Executable, [Parameter(Mandatory)][string]$OutputDirectory, [Parameter(Mandatory)][string]$RecoveryProbe)
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true') { throw 'This destructive service lifecycle test is restricted to disposable CI runners.' }
if (Get-Service -Name 'TorrServer-Flow' -ErrorAction SilentlyContinue) { throw 'A service already exists; refusing to touch it.' }
$exe = (Resolve-Path -LiteralPath $Executable).Path
$recoveryProbePath = (Resolve-Path -LiteralPath $RecoveryProbe).Path
$state = [IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $state) { throw 'Test state must be new.' }
[void](New-Item -ItemType Directory -Path $state)
$reservation = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0)
$reservation.Start(); $port = $reservation.LocalEndpoint.Port; $reservation.Stop()
@{ fixture = 'service-fixture-only' } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $state 'accs.db') -Encoding ASCII
@{ BitTorr = @{ DisableDHT=$true; DisableUPNP=$true; DisablePEX=$true; EnableBonjour=$false; EnableDLNA=$false; RetrackersMode=0; DefaultTrackers='' } } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $state 'settings.json') -Encoding ASCII
$headers = @{ Authorization='Basic '+[Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes('fixture:service-fixture-only')) }
$baseUri = 'http://127.0.0.1:'+$port
function Command([string[]]$Arguments) {
    & $exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Service command failed: $($Arguments -join ' ')" }
}
function Health {
    $deadline = [DateTime]::UtcNow.AddSeconds(40)
    do {
        try {
            $status = Invoke-RestMethod -Uri "$baseUri/flow/tray" -Headers $headers -TimeoutSec 2
            if ($status.server_state -eq 'RUNNING') { return }
        } catch { }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw 'Restricted service did not become ready.'
}
$installed = $false
try {
    Command -Arguments @('--service','install','--path',$state,'--port',[string]$port,'--ip','127.0.0.1','--httpauth')
    $installed = $true
    $service = Get-CimInstance Win32_Service -Filter "Name='TorrServer-Flow'"
    if ($service.StartName -ne 'NT SERVICE\TorrServer-Flow') { throw 'Service is not using its virtual account.' }
    $sidType = (Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Services\TorrServer-Flow').ServiceSidType
    if ($sidType -ne 3) { throw 'Service SID is not restricted.' }
    $failurePolicy = (& sc.exe qfailure TorrServer-Flow | Out-String)
    $failurePolicy | Set-Content -LiteralPath (Join-Path $state 'recovery-display.txt')
    if ($LASTEXITCODE -ne 0) { throw 'Cannot read service recovery policy.' }
    $policy = (& $recoveryProbePath | Out-String)
    $policy | Set-Content -LiteralPath (Join-Path $state 'recovery-policy.json')
    if ($LASTEXITCODE -ne 0) { throw "Bounded recovery policy mismatch: $policy" }
    Command -Arguments @('--service','start'); Health
    $crashedPID = (Get-CimInstance Win32_Service -Filter "Name='TorrServer-Flow'").ProcessId
    if ($crashedPID -le 0) { throw 'Fixture service PID missing.' }
    Stop-Process -Id $crashedPID -Force
    $deadline = [DateTime]::UtcNow.AddSeconds(45)
    do {
        Start-Sleep -Milliseconds 500
        $recovered = Get-CimInstance Win32_Service -Filter "Name='TorrServer-Flow'"
    } while (($recovered.ProcessId -eq 0 -or $recovered.ProcessId -eq $crashedPID) -and [DateTime]::UtcNow -lt $deadline)
    if ($recovered.ProcessId -eq 0 -or $recovered.ProcessId -eq $crashedPID) { throw 'SCM crash recovery failed.' }
    Health
    $lease = Invoke-RestMethod -Uri "$baseUri/flow/maintenance" -Method Post -ContentType 'application/json' -Body '{"enabled":true}' -Headers $headers
    if (-not $lease.token) { throw 'Maintenance lease missing.' }
    Invoke-RestMethod -Uri "$baseUri/flow/maintenance" -Method Post -ContentType 'application/json' -Body (@{enabled=$false;token=$lease.token}|ConvertTo-Json -Compress) -Headers $headers | Out-Null
    Command -Arguments @('--service','restart'); Health
    $report = Invoke-RestMethod -Uri "$baseUri/flow/support" -Headers $headers
    if (($report|ConvertTo-Json -Depth 20) -match 'service-fixture-only') { throw 'Support report leaked an account.' }
    Command -Arguments @('--service','stop')
    if ((Get-Service -Name 'TorrServer-Flow').Status -ne 'Stopped') { throw 'Service did not stop.' }
    @{passed=$true;restricted_account=$true;restricted_sid=$true;restart=$true;clean_stop=$true;bounded_crash_policy=$true;crash_recovery=$true;path=$state} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $state 'service-test.json')
} finally {
    if ($installed) { Command -Arguments @('--service','uninstall') }
}
