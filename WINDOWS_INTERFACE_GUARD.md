# Optional Windows interface guard

`FlowNetworkGuard-windows-amd64.exe` creates an explicit Windows Filtering
Platform (WFP) policy for the executable path you select. It is separate from
Flow's source-interface binding. Nothing installs or elevates it automatically.

## Scope and limits

- The selected executable's new IPv4/IPv6 TCP and UDP connections use the
  selected interface. Public connections on other interfaces are blocked.
- Local/private addresses, loopback, IPv4 link-local and local discovery
  multicast remain permitted so the phone can reach the server. Existing
  Windows firewall restrictions still apply. LAN proxies and private-address
  destinations are deliberately outside the public-address restriction.
- Rules are persistent, apply to this **path**, and survive helper exit and
  reboot. Moving or renaming Flow requires removing the old policy and
  installing a policy for the new path. Replacing the binary at the same path
  retains the policy. Other executables and system services are unaffected.
- This also restricts other Internet operations performed by that executable,
  including updates and searches. Already-open connections should be closed
  before installation; start Flow after installation to test new connections.
- The guard does **not** filter shared Windows DNS services. Interface binding
  and an installed filter set alone do not prove DNS or complete leak
  protection. Both verification flags remain false. Physical adapter loss,
  reconnect, VPN routing and protocol tests remain separate acceptance checks.

Use it only after inspecting the preview and understanding these limits.
Installation/removal require an Administrator terminal; preview does not.

## Preview, install, inspect, undo

Find your intended adapter's `ifIndex` with `Get-NetAdapter`. Replace `N` below
with that number and replace the executable path with the one you launch.

```powershell
.\FlowNetworkGuard-windows-amd64.exe preview --exe 'C:\Flow\TorrServer-LT-windows-amd64.exe' --interface-index N
.\FlowNetworkGuard-windows-amd64.exe install --exe 'C:\Flow\TorrServer-LT-windows-amd64.exe' --interface-index N
.\FlowNetworkGuard-windows-amd64.exe status --exe 'C:\Flow\TorrServer-LT-windows-amd64.exe'
.\FlowNetworkGuard-windows-amd64.exe remove --exe 'C:\Flow\TorrServer-LT-windows-amd64.exe'
```

Installation replaces only this helper's owned filters for the exact executable
in one WFP transaction. Failure rolls back that transaction. Repeated install
does not accumulate duplicate filters. Remove leaves other executable policies
intact. If the old executable was deleted and its path cannot be resolved,
`remove-all-owned` explicitly removes **all** policies owned by this helper;
it does not delete unrelated Windows firewall filters.

Status reports installed filter count and configuration completeness. It is
not a packet capture or a claim that every protocol has been verified. Error
output contains Windows error numbers, without executable paths or credentials.

## Build and test

Production builds use the same MinGW-w64/GCC runtime and original runtime
notices as Flow's Windows native build. The release records helper binary and
source SHA-256 values, source commit and provenance. The source is covered by
the repository's GPL-3.0 license.

Local MSVC/Windows SDK compilation and preview checks:

```powershell
.\build\build-windows-interface-guard.ps1
.\build\test-windows-interface-guard.ps1 -Guard .tools\windows-interface-guard\FlowNetworkGuard-windows-amd64.exe
```

The `-InstallFixture` test switch installs filters for generated probe binaries
only. CI runs it on a disposable Windows runner, verifies loopback playback
connections, blocked public fallback, repeated installation, removal and
isolation between two executable paths. IPv6 without a usable route remains
explicitly unverified. Do not run that switch on a normal workstation merely
to view a preview. It is never invoked by the server or installer.

WFP uses Microsoft's documented [application and interface conditions](https://learn.microsoft.com/en-us/windows/win32/fwp/filtering-condition-identifiers-)
and [filter arbitration](https://learn.microsoft.com/en-us/windows/win32/fwp/filter-arbitration).
Selected-interface/LAN permits have higher weights within the helper's own
sublayer than its fallback blocks; restrictions in other firewall sublayers
can still block connections.
