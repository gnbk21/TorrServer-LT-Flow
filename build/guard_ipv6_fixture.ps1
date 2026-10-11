# Disposable CI-only documentation-address route for kernel IPv6 classification.
# Never sourced by the server, production helper or installation scripts.
function Assert-DisposableGuardRunner {
  if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted' -or $env:FLOW_ISOLATED_WFP -ne 'github-hosted') {
    throw 'Network fixture requires an explicitly selected disposable GitHub-hosted runner'
  }
}
function New-GuardIPv6Fixture {
  param([ValidatePattern('^[a-f0-9]{4}$')][string]$AddressSuffix = ([Guid]::NewGuid().ToString('N').Substring(0,4)))
  Assert-DisposableGuardRunner
  $adapter = Get-NetIPInterface -AddressFamily IPv6 -ConnectionState Connected |
    Where-Object { $_.InterfaceIndex -ne $script:guardLoopbackIndex } |
    Sort-Object InterfaceMetric | Select-Object -First 1
  if (!$adapter) { throw 'Disposable IPv6 fixture requires a connected non-loopback interface' }
  $fixture = [pscustomobject]@{
    InterfaceIndex = $adapter.InterfaceIndex
    # NetTCPIP stores a canonical address. Querying a generated leading-zero
    # spelling (e.g. ::05d3) can fail to match its stored ::5d3 representation.
    Address = ([Net.IPAddress]::Parse('2001:db8:ffff::' + $AddressSuffix)).ToString()
    Prefix = '2001:db8::1/128'
    OriginalDhcp = $adapter.Dhcp
    AddressAttempted = $false
    RouteAttempted = $false
  }
  try {
    if (Get-NetIPAddress -IPAddress $fixture.Address -ErrorAction SilentlyContinue) { throw 'IPv6 fixture address already exists' }
    if (Get-NetRoute -InterfaceIndex $fixture.InterfaceIndex -DestinationPrefix $fixture.Prefix -ErrorAction SilentlyContinue) { throw 'IPv6 fixture route already exists' }
    $fixture.AddressAttempted = $true
    New-NetIPAddress -InterfaceIndex $fixture.InterfaceIndex -IPAddress $fixture.Address -AddressFamily IPv6 -PrefixLength 128 -SkipAsSource $true -PolicyStore ActiveStore | Out-Null
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    do {
      $address = Get-NetIPAddress -InterfaceIndex $fixture.InterfaceIndex -IPAddress $fixture.Address -AddressFamily IPv6
      if ($address.AddressState -eq 'Preferred') { break }
      if ([DateTime]::UtcNow -ge $deadline) { throw 'IPv6 fixture address did not become usable' }
      Start-Sleep -Milliseconds 200
    } while ($true)
    $fixture.RouteAttempted = $true
    New-NetRoute -InterfaceIndex $fixture.InterfaceIndex -DestinationPrefix $fixture.Prefix -NextHop '::' -PolicyStore ActiveStore -RouteMetric 1 | Out-Null
    return $fixture
  } catch {
    Remove-GuardIPv6Fixture $fixture
    throw
  }
}
function Remove-GuardIPv6Fixture($Fixture) {
  Assert-DisposableGuardRunner
  if (!$Fixture) { return }
  $failures = @()
  if ($Fixture.RouteAttempted) {
    try {
      Get-NetRoute -InterfaceIndex $Fixture.InterfaceIndex -DestinationPrefix $Fixture.Prefix -NextHop '::' -PolicyStore ActiveStore -ErrorAction SilentlyContinue |
        Remove-NetRoute -Confirm:$false -ErrorAction Stop
      if (Get-NetRoute -InterfaceIndex $Fixture.InterfaceIndex -DestinationPrefix $Fixture.Prefix -NextHop '::' -PolicyStore ActiveStore -ErrorAction SilentlyContinue) {
        throw 'Owned route remains'
      }
    } catch { $failures += 'Owned IPv6 fixture route cleanup failed' }
  }
  if ($Fixture.AddressAttempted) {
    try {
      Get-NetIPAddress -InterfaceIndex $Fixture.InterfaceIndex -IPAddress $Fixture.Address -AddressFamily IPv6 -PolicyStore ActiveStore -ErrorAction SilentlyContinue |
        Remove-NetIPAddress -Confirm:$false -ErrorAction Stop
      if (Get-NetIPAddress -InterfaceIndex $Fixture.InterfaceIndex -IPAddress $Fixture.Address -AddressFamily IPv6 -PolicyStore ActiveStore -ErrorAction SilentlyContinue) {
        throw 'Owned address remains'
      }
    } catch { $failures += 'Owned IPv6 fixture address cleanup failed' }
    try {
      $current = Get-NetIPInterface -InterfaceIndex $Fixture.InterfaceIndex -AddressFamily IPv6
      if ($current.Dhcp -ne $Fixture.OriginalDhcp) {
        Set-NetIPInterface -InterfaceIndex $Fixture.InterfaceIndex -AddressFamily IPv6 -Dhcp $Fixture.OriginalDhcp -PolicyStore ActiveStore
      }
      if ((Get-NetIPInterface -InterfaceIndex $Fixture.InterfaceIndex -AddressFamily IPv6).Dhcp -ne $Fixture.OriginalDhcp) { throw 'DHCP restoration did not converge' }
    } catch { $failures += 'Disposable IPv6 DHCP setting restoration failed' }
  }
  if ($failures.Count) { throw ($failures -join '; ') }
}
