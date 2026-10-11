# Portable lifecycle tests with mock cmdlets; no operating-system policy changes.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'guard_ipv6_fixture.ps1')
$script:guardLoopbackIndex = 1
$script:fixtureState = @{ address=$null; route=$null; dhcp='Enabled'; calls=0; failAddress=$false; failRoute=$false; collision=$false }
function Get-NetIPInterface {
  [CmdletBinding()]param($AddressFamily,$ConnectionState,$InterfaceIndex)
  $script:fixtureState.calls++
  [pscustomobject]@{ InterfaceIndex=17; InterfaceMetric=10; Dhcp=$script:fixtureState.dhcp }
}
function Get-NetIPAddress {
  [CmdletBinding()]param($IPAddress,$InterfaceIndex,$AddressFamily,$PolicyStore)
  if ($script:fixtureState.address -eq $IPAddress) {
    [pscustomobject]@{ IPAddress=$IPAddress; InterfaceIndex=17; AddressState='Preferred' }
  }
}
function Get-NetRoute {
  [CmdletBinding()]param($InterfaceIndex,$DestinationPrefix,$NextHop,$PolicyStore)
  if ($script:fixtureState.collision -or $script:fixtureState.route) {
    [pscustomobject]@{ DestinationPrefix='2001:db8::1/128'; InterfaceIndex=17 }
  }
}
function New-NetIPAddress {
  [CmdletBinding()]param($InterfaceIndex,$IPAddress,$AddressFamily,$PrefixLength,$SkipAsSource,$PolicyStore)
  if ($InterfaceIndex -ne 17 -or $AddressFamily -ne 'IPv6' -or $PrefixLength -ne 128 -or !$SkipAsSource -or $PolicyStore -ne 'ActiveStore') { throw 'Unexpected fixture address scope' }
  $script:fixtureState.address=$IPAddress
  $script:fixtureState.dhcp='Disabled'
  if ($script:fixtureState.failAddress) { throw 'Injected partial address failure' }
}
function New-NetRoute {
  [CmdletBinding()]param($InterfaceIndex,$DestinationPrefix,$NextHop,$PolicyStore,$RouteMetric)
  if ($InterfaceIndex -ne 17 -or $DestinationPrefix -ne '2001:db8::1/128' -or $NextHop -ne '::' -or $PolicyStore -ne 'ActiveStore' -or $RouteMetric -ne 1) { throw 'Unexpected fixture route scope' }
  $script:fixtureState.route=$DestinationPrefix
  if ($script:fixtureState.failRoute) { throw 'Injected partial route failure' }
}
function Remove-NetIPAddress {
  [CmdletBinding(SupportsShouldProcess)]param([Parameter(ValueFromPipeline)]$InputObject)
  process { $script:fixtureState.address=$null }
}
function Remove-NetRoute {
  [CmdletBinding(SupportsShouldProcess)]param([Parameter(ValueFromPipeline)]$InputObject)
  process { $script:fixtureState.route=$null }
}
function Set-NetIPInterface {
  [CmdletBinding()]param($InterfaceIndex,$AddressFamily,$Dhcp,$PolicyStore)
  if ($InterfaceIndex -ne 17 -or $AddressFamily -ne 'IPv6' -or $PolicyStore -ne 'ActiveStore') { throw 'Unexpected DHCP restoration scope' }
  $script:fixtureState.dhcp=$Dhcp
}
function Assert-FixtureClean {
  if ($script:fixtureState.address -or $script:fixtureState.route -or $script:fixtureState.dhcp -ne 'Enabled') { throw 'Fixture state was not restored' }
}
$saved = @{}
foreach ($key in @('GITHUB_ACTIONS','RUNNER_ENVIRONMENT','FLOW_ISOLATED_WFP')) { $saved[$key]=[Environment]::GetEnvironmentVariable($key) }
try {
  $env:GITHUB_ACTIONS='false'
  $rejected=$false
  try { New-GuardIPv6Fixture | Out-Null } catch { $rejected=$true }
  if (!$rejected -or $script:fixtureState.calls) { throw 'Ordinary workstation fixture was not rejected before network access' }
  $env:GITHUB_ACTIONS='true'; $env:RUNNER_ENVIRONMENT='github-hosted'; $env:FLOW_ISOLATED_WFP='github-hosted'
  $owned=New-GuardIPv6Fixture -AddressSuffix '05d3'
  if ($owned.Address -ne '2001:db8:ffff::5d3') { throw 'Leading-zero IPv6 fixture address was not canonicalized before CIM lookup' }
  if (!$script:fixtureState.address -or !$script:fixtureState.route) { throw 'Fixture was not created' }
  Remove-GuardIPv6Fixture $owned
  Assert-FixtureClean
  foreach ($fault in @('failAddress','failRoute')) {
    $script:fixtureState[$fault]=$true
    $rejected=$false
    try { New-GuardIPv6Fixture | Out-Null } catch { $rejected=$true }
    if (!$rejected) { throw 'Partial fixture creation was not rejected' }
    Assert-FixtureClean
    $script:fixtureState[$fault]=$false
  }
  $script:fixtureState.collision=$true
  $rejected=$false
  try { New-GuardIPv6Fixture | Out-Null } catch { $rejected=$true }
  if (!$rejected -or !$script:fixtureState.collision) { throw 'Existing route was not preserved' }
  Assert-FixtureClean
  Write-Output 'Disposable IPv6 fixture admission, ownership, rollback and cleanup checks passed.'
} finally {
  foreach ($key in $saved.Keys) { [Environment]::SetEnvironmentVariable($key,$saved[$key]) }
}
