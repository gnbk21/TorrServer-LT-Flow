param(
  [Parameter(Mandatory)][string]$Guard,
  [string]$OutputDirectory = '.tools/windows-interface-guard-check',
  [switch]$InstallFixture
)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
. (Join-Path $PSScriptRoot 'guard_ipv6_fixture.ps1')
$output = [IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDirectory))
if (!$output.StartsWith($repoRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Test output must be inside the checkout' }
New-Item -ItemType Directory -Path $output -Force | Out-Null
$guardPath = (Resolve-Path -LiteralPath $Guard).Path
function Invoke-Guard([string[]]$Arguments) {
  $text = & $guardPath @Arguments
  if ($LASTEXITCODE -ne 0) { throw "Guard command failed: $LASTEXITCODE" }
  return ($text | ConvertFrom-Json)
}
$loopback = [Net.NetworkInformation.NetworkInterface]::GetAllNetworkInterfaces() | Where-Object NetworkInterfaceType -EQ Loopback | Select-Object -First 1
if (!$loopback) { throw 'Loopback fixture interface is unavailable' }
$index = $loopback.GetIPProperties().GetIPv4Properties().Index
$script:guardLoopbackIndex = $index
$preview = Invoke-Guard -Arguments @('preview','--exe',$guardPath,'--interface-index',"$index")
if ($preview.changes_applied -or $preview.planned_filters -ne 30 -or $preview.dns_verified -or $preview.leak_protection_verified) { throw 'Preview modified policy or claimed unverified protection' }
$report = [ordered]@{ preview=$preview; installation_tested=$false; scope='Isolated generated fixture executable only; no system DNS rules'; protocols=@{} }
if ($InstallFixture) {
  if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted' -or $env:FLOW_ISOLATED_WFP -ne 'github-hosted') { throw 'Fixture installation requires an explicitly selected disposable GitHub-hosted runner' }
  # This switch is used only on the disposable Windows CI runner. Never called
  # automatically by installers or a running user's server.
  $fixture = Join-Path $output 'guard-fixture.exe'
  $second = Join-Path $output 'guard-second-fixture.exe'
  & go build -o $fixture (Join-Path $PSScriptRoot 'guard_probe_windows.go')
  if ($LASTEXITCODE -ne 0) { throw 'Fixture compilation failed' }
  Copy-Item -LiteralPath $fixture -Destination $second
  & (Join-Path $PSScriptRoot 'build-windows-interface-guard.ps1') -OutputDirectory $OutputDirectory -BuildAcceptanceProbe | Out-Host
  $dropProbe = Join-Path $output 'guard-drop-probe.exe'
  foreach ($app in @($fixture,$second)) {
    if ((Invoke-Guard -Arguments @('status','--exe',$app)).installed) { throw 'Fresh fixture unexpectedly has installed filters' }
  }
  $ipv6Fixture = $null
  try {
    $ipv6Fixture = New-GuardIPv6Fixture
    $fixtureArguments = @('--ipv6-source', $ipv6Fixture.Address)
    $report.ipv6_fixture = 'Owned ActiveStore documentation-address source and /128 on-link route; no default-route or DNS changes'
    $baseline = & $fixture @fixtureArguments | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) { throw 'Baseline fixture failed' }
    foreach ($app in @($fixture,$second)) {
      $installed = Invoke-Guard -Arguments @('install','--exe',$app,'--interface-index',"$index")
      if (!$installed.configuration_complete -or $installed.filter_count -ne 30) { throw 'Atomic policy installation is incomplete' }
    }
    $repeated = Invoke-Guard -Arguments @('install','--exe',$fixture,'--interface-index',"$index")
    if ($repeated.filter_count -ne 30) { throw 'Repeated install duplicated filters' }
    $startedAt = [DateTime]::UtcNow.ToFileTimeUtc()
    $protected = & $fixture @fixtureArguments | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) { throw 'Installed fixture failed' }
    $drops = $null
    for ($attempt = 0; $attempt -lt 10; $attempt++) {
      Start-Sleep -Milliseconds 200
      $drops = & $dropProbe $fixture "$startedAt" | ConvertFrom-Json
      if ($LASTEXITCODE -ne 0) { throw 'Scoped kernel drop evidence is unavailable' }
      if ($drops.tcp4 -gt 0 -and $drops.udp4 -gt 0 -and $drops.tcp6 -gt 0 -and $drops.udp6 -gt 0) { break }
    }
    foreach ($protocol in @('tcp4','udp4','tcp6','udp6')) {
      $local = "lan_$protocol"
      $remote = "public_$protocol"
      if (!$baseline.$local.success -or !$protected.$local.success) { throw "LAN $protocol was not verified" }
      $eligible = !$baseline.$remote.access_denied -and !$baseline.$remote.unavailable
      # UDP Write may report queued success even when WFP drops the first packet.
      # Require kernel evidence from this exact app and an owned Flow block rule.
      $verified = $eligible -and $drops.$protocol -gt 0
      $report.protocols[$protocol] = @{ verified=$verified; owned_classify_drops=$drops.$protocol; evidence='Exact fixture/app/time/destination and owned WFP block rule'; baseline=$baseline.$remote; installed=$protected.$remote; lan=$protected.$local }
      if (!$verified) { throw "$protocol fallback enforcement was not verified" }
    }
    Invoke-Guard -Arguments @('remove','--exe',$fixture) | Out-Null
    if ((Invoke-Guard -Arguments @('status','--exe',$fixture)).installed) { throw 'Removal retained fixture policy' }
    if (!(Invoke-Guard -Arguments @('status','--exe',$second)).configuration_complete) { throw 'Removal affected another executable' }
    $restored = & $fixture @fixtureArguments | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) { throw 'Restored fixture failed' }
    foreach ($protocol in @('tcp4','udp4','tcp6','udp6')) {
      if ($restored."public_$protocol".access_denied -or $restored."public_$protocol".unavailable) { throw "Removal did not restore $protocol routing" }
    }
    $report.installation_tested = $true
    $report.removal_restored_routing = $true
    $report.other_executable_preserved = $true
  } catch {
    $report.failure = 'Isolated WFP fixture acceptance failed; inspect CI output for the failing gate'
    throw
  } finally {
    $cleanupErrors = @()
    foreach ($app in @($fixture,$second)) {
      try { Invoke-Guard -Arguments @('remove','--exe',$app) | Out-Null }
      catch { $cleanupErrors += 'An owned fixture policy could not be removed' }
    }
    try { Remove-GuardIPv6Fixture $ipv6Fixture }
    catch { $cleanupErrors += 'An owned IPv6 fixture could not be completely removed' }
    $report.cleanup_errors = $cleanupErrors
    $report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $output 'report.json') -Encoding utf8
    if ($cleanupErrors.Count) { throw ($cleanupErrors -join '; ') }
  }
}
$report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $output 'report.json') -Encoding utf8
Write-Output 'Windows interface guard checks passed; report contains the tested scope.'
