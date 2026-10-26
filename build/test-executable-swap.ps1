$ErrorActionPreference='Stop'
. (Join-Path (Split-Path $PSScriptRoot -Parent) 'distribution/FlowRelease.ps1')
$script:calls=0
function Move-Item {
    param([string]$LiteralPath,[string]$Destination,[string]$ErrorAction)
    $script:calls++
    if ($script:calls -le 2) { throw [IO.IOException]::new('sharing fixture', [int]0x80070020) }
    if ($LiteralPath -ne 'owned-fixture.exe' -or $Destination -ne 'recovery-fixture.exe') { throw 'Paths changed.' }
}
Move-FlowExecutable 'owned-fixture.exe' 'recovery-fixture.exe'
if ($script:calls -ne 3) { throw 'Transient sharing violations were not retried.' }
function Move-Item {
    param([string]$LiteralPath,[string]$Destination,[string]$ErrorAction)
    $script:calls++
    throw [UnauthorizedAccessException]::new('permission fixture')
}
$script:calls=0; $rejected=$false
try { Move-FlowExecutable 'owned-fixture.exe' 'recovery-fixture.exe' } catch { $rejected=$true }
if (-not $rejected -or $script:calls -ne 1) { throw 'Permission failure was swallowed or retried.' }
Write-Output 'Executable sharing retry and permission rejection passed.'
