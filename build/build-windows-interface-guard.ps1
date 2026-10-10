param([string]$OutputDirectory = '.tools/windows-interface-guard', [switch]$BuildAcceptanceProbe)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$output = [IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDirectory))
if (!$output.StartsWith($repoRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Build output must be inside the checkout' }
New-Item -ItemType Directory -Path $output -Force | Out-Null
$vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
$visualStudio = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
if (!$visualStudio) { throw 'Install the Visual Studio C++ build tools and Windows 10/11 SDK' }
$version = (Get-Content -LiteralPath (Join-Path $visualStudio 'VC/Auxiliary/Build/Microsoft.VCToolsVersion.default.txt') -Raw).Trim()
$vc = Join-Path $visualStudio "VC/Tools/MSVC/$version"
$sdk = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits/10'
$sdkVersion = (Get-ChildItem -LiteralPath (Join-Path $sdk 'Include') -Directory | Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName 'um/fwpmu.h') } | Sort-Object { [version]$_.Name } -Descending | Select-Object -First 1).Name
if (!$sdkVersion) { throw 'Windows SDK WFP headers are unavailable' }
$oldInclude = $env:INCLUDE
$oldLib = $env:LIB
try {
  $env:INCLUDE = "$vc/include;$sdk/Include/$sdkVersion/ucrt;$sdk/Include/$sdkVersion/shared;$sdk/Include/$sdkVersion/um"
  $env:LIB = "$vc/lib/x64;$sdk/Lib/$sdkVersion/ucrt/x64;$sdk/Lib/$sdkVersion/um/x64"
  & (Join-Path $vc 'bin/Hostx64/x64/cl.exe') /nologo /EHsc /std:c++17 /W4 /WX /MT /D_WIN32_WINNT=0x0A00 /DUNICODE /D_UNICODE "/Fe:$output/FlowNetworkGuard-windows-amd64.exe" "/Fo:$output/guard.obj" (Join-Path $PSScriptRoot 'windows-interface-guard.cpp') /link fwpuclnt.lib iphlpapi.lib advapi32.lib ole32.lib ws2_32.lib
  if ($LASTEXITCODE -ne 0) { throw "WFP helper compilation failed: $LASTEXITCODE" }
  if ($BuildAcceptanceProbe) {
    & (Join-Path $vc 'bin/Hostx64/x64/cl.exe') /nologo /EHsc /std:c++17 /W4 /WX /MT /D_WIN32_WINNT=0x0A00 /DUNICODE /D_UNICODE "/Fe:$output/guard-drop-probe.exe" "/Fo:$output/drop-probe.obj" (Join-Path $PSScriptRoot 'guard_drop_probe_windows.cpp') /link fwpuclnt.lib ws2_32.lib
    if ($LASTEXITCODE -ne 0) { throw "CI drop probe compilation failed: $LASTEXITCODE" }
  }
} finally {
  $env:INCLUDE = $oldInclude
  $env:LIB = $oldLib
}
