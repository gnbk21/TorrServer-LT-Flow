# Isolated filesystem recovery: no installation, process or network operations.
$ErrorActionPreference = 'Stop'
. (Join-Path (Split-Path $PSScriptRoot -Parent) 'distribution/FlowRelease.ps1')
$root = Join-Path ([IO.Path]::GetTempPath()) ('flow-state-test-'+[guid]::NewGuid().ToString('N'))
$state=Join-Path $root 'shared'; $backup=Join-Path $root 'backup'
[void](New-Item -ItemType Directory -Path $state,$backup)
try {
    'original' | Set-Content -LiteralPath (Join-Path $state 'settings.json')
    'unrelated' | Set-Content -LiteralPath (Join-Path $state 'notes.txt')
    Save-FlowStateSnapshot $state $backup
    if (Test-Path -LiteralPath (Join-Path $backup 'notes.txt')) { throw 'Unrelated file copied.' }
    'changed' | Set-Content -LiteralPath (Join-Path $state 'settings.json')
    'new user file' | Set-Content -LiteralPath (Join-Path $state 'new-document.json')
    'new Flow file' | Set-Content -LiteralPath (Join-Path $state 'flow-settings-pending.json')
    Restore-FlowStateSnapshot $state $backup
    if ((Get-Content -LiteralPath (Join-Path $state 'settings.json')).Trim() -ne 'original') { throw 'Original not restored.' }
    foreach($name in @('notes.txt','new-document.json')) { if (-not (Test-Path -LiteralPath (Join-Path $state $name))) { throw 'Unrelated file moved.' } }
    if (Test-Path -LiteralPath (Join-Path $state 'flow-settings-pending.json')) { throw 'New recognized state not quarantined.' }
    if (-not (Test-Path -LiteralPath (Join-Path $backup 'failed-state/flow-settings-pending.json'))) { throw 'Recognized state lost.' }
    $rejected=$false
    try { Save-FlowStateSnapshot $state $backup } catch { $rejected=$true }
    if (-not $rejected) { throw 'Existing recovery inventory was replaced.' }
    foreach($name in @('../notes.txt','C:/notes.txt','ssl/pair-../uploaded.key','other.json')) {
        $rejected=$false
        try { [void](Resolve-FlowStateFile $state $name) } catch { $rejected=$true }
        if (-not $rejected) { throw 'Unsafe inventory entry accepted.' }
    }
    if ([Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT) {
    $outside=Join-Path $root 'outside'
    [void](New-Item -ItemType Directory -Path $outside)
    [void](New-Item -ItemType Junction -Path (Join-Path $state 'ssl') -Target $outside)
    $rejected=$false
    try { [void](Get-FlowOwnedState $state) } catch { $rejected=$true }
    Remove-Item -LiteralPath (Join-Path $state 'ssl') -Force
    if (-not $rejected) { throw 'A junction was followed.' }
    $backupLink=Join-Path $root 'backup-link'
    [void](New-Item -ItemType Junction -Path $backupLink -Target $outside)
    $rejected=$false
    try { Save-FlowStateSnapshot $state (Join-Path $backupLink 'nested') } catch { $rejected=$true }
    Remove-Item -LiteralPath $backupLink -Force
    if (-not $rejected -or (Test-Path -LiteralPath (Join-Path $outside 'nested'))) { throw 'A backup ancestor junction was followed.' }
    $linked=Join-Path $outside 'private.txt'
    'external fixture' | Set-Content -LiteralPath $linked
    Remove-Item -LiteralPath (Join-Path $state 'settings.json')
    [void](New-Item -ItemType HardLink -Path (Join-Path $state 'settings.json') -Target $linked)
    $rejected=$false
    try { [void](Get-FlowOwnedState $state) } catch { $rejected=$true }
    if (-not $rejected) { throw 'A multiply linked state file was accepted.' }
    Remove-Item -LiteralPath (Join-Path $state 'settings.json')
    'original' | Set-Content -LiteralPath (Join-Path $state 'settings.json')
    }
    'tampered' | Set-Content -LiteralPath (Join-Path $backup 'settings.json')
    $rejected=$false
    try { Restore-FlowStateSnapshot $state $backup } catch { $rejected=$true }
    if (-not $rejected) { throw 'Tampered recovery accepted.' }
    Write-Output 'Owned-state recovery and shared-directory preservation passed.'
} finally {
    $target=[IO.Path]::GetFullPath($root)
    $allowed=[IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\','/')+[IO.Path]::DirectorySeparatorChar
    if (-not $target.StartsWith($allowed,[StringComparison]::OrdinalIgnoreCase) -or (Split-Path $target -Leaf) -notlike 'flow-state-test-*') { throw 'Unsafe test cleanup target.' }
    Remove-Item -LiteralPath $target -Recurse -Force
}
