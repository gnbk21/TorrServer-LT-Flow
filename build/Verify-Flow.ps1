[CmdletBinding()]
param([ValidateSet('Quick','Full')][string]$Scope = 'Quick')
$ErrorActionPreference = 'Stop'
$flowVerifyRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $flowVerifyRoot
. "$PSScriptRoot/Enter-FlowBuild.ps1"
$flowVerifyOutput = Join-Path $flowVerifyRoot ('_out/verification/' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfff'))
[void](New-Item -ItemType Directory -Path $flowVerifyOutput)
$flowVerifySteps = [Collections.Generic.List[object]]::new()
$flowVerifyReceipt = [ordered]@{
    schema_version = 1; scope = $Scope; passed = $false
    source = (& git rev-parse HEAD).Trim()
    modified = [bool](& git status --porcelain)
    started_at = [DateTime]::UtcNow.ToString('o')
    go_version = (& go version); steps = $flowVerifySteps
    field_acceptance = 'Controlled checks only; phone and public-swarm acceptance remain separate.'
}
function Invoke-FlowVerificationStep {
    param([string]$Name, [string]$Program, [string[]]$Arguments, [string]$Directory = $flowVerifyRoot)
    $flowStepTimer = [Diagnostics.Stopwatch]::StartNew()
    $flowStepLog = Join-Path $flowVerifyOutput ($Name + '.log')
    Push-Location -LiteralPath $Directory
    try {
        & $Program @Arguments *> $flowStepLog
        $flowStepExit = $LASTEXITCODE
    } finally { Pop-Location }
    $flowVerifySteps.Add([ordered]@{name=$Name;exit_code=$flowStepExit;elapsed_ms=$flowStepTimer.ElapsedMilliseconds;log=[IO.Path]::GetFileName($flowStepLog)})
    if ($flowStepExit -ne 0) { throw "$Name failed; see $flowStepLog" }
    Write-Host ("{0}: passed ({1:N1}s)" -f $Name,$flowStepTimer.Elapsed.TotalSeconds)
}
try {
    $flowNode = (Get-Command node.exe -ErrorAction Stop).Source
    $flowPython = if (Test-Path -LiteralPath 'C:/Python314/python.exe') { 'C:/Python314/python.exe' } else { (Get-Command python.exe -ErrorAction Stop).Source }
    $flowGo = (Get-Command go.exe -ErrorAction Stop).Source
    $flowWeb = Join-Path $flowVerifyRoot 'web'
    if (!(Test-Path -LiteralPath (Join-Path $flowWeb 'node_modules/typescript/bin/tsc'))) { throw 'Install locked web dependencies with yarn install --frozen-lockfile first.' }
    Invoke-FlowVerificationStep 'package-inventory' $flowPython @('-X','utf8','build/test_packages.py')
    Invoke-FlowVerificationStep 'harness-tests' $flowPython @('-X','utf8','-m','unittest','discover','-s','build','-p','*_test.py')
    Invoke-FlowVerificationStep 'web-lint' $flowNode @('node_modules/eslint/bin/eslint.js','src') $flowWeb
    Invoke-FlowVerificationStep 'web-tooling-contract' $flowNode @('--test','scripts/linter-glob-compat.test.mjs') $flowWeb
    Invoke-FlowVerificationStep 'web-tests' $flowNode @('node_modules/vitest/vitest.mjs','run') $flowWeb
    Invoke-FlowVerificationStep 'web-typecheck' $flowNode @('node_modules/typescript/bin/tsc','--noEmit') $flowWeb
    Invoke-FlowVerificationStep 'web-build' $flowNode @('node_modules/vite/bin/vite.js','build') $flowWeb
    Invoke-FlowVerificationStep 'web-notices' $flowNode @('scripts/notices.mjs') $flowWeb
    Invoke-FlowVerificationStep 'embedding-tests' $flowGo @('test','gen_web.go','gen_web_test.go')
    Invoke-FlowVerificationStep 'web-embedding' $flowGo @('run','gen_web.go')
    Invoke-FlowVerificationStep 'native-build' (Join-Path $flowVerifyRoot '.tools/msys64/usr/bin/bash.exe') @('-lc','JOBS=4 bash build/windows-native.sh')
    # The native build may have replaced its static archive. Refresh CGo's
    # cache stamp before unit tests consume the verified installation.
    . "$PSScriptRoot/Enter-FlowBuild.ps1"
    $flowPackages = (& $flowPython -X utf8 build/test_packages.py --list native).Split(' ')
    if ($LASTEXITCODE) { throw 'Package inventory changed during verification' }
    Invoke-FlowVerificationStep 'native-unit' $flowGo (@('test','-p','2','-json','-count=1','-timeout','180s')+$flowPackages) (Join-Path $flowVerifyRoot 'server')
    Invoke-FlowVerificationStep 'native-vet' $flowGo (@('vet')+$flowPackages) (Join-Path $flowVerifyRoot 'server')
    if ($Scope -eq 'Full') {
        Invoke-FlowVerificationStep 'native-race' $flowGo (@('test','-p','2','-race','-count=1','-timeout','180s')+$flowPackages) (Join-Path $flowVerifyRoot 'server')
        $flowPreviousCI = $env:CI
        try {
            $env:CI = '1' # Never reuse another project's listener for acceptance.
            Invoke-FlowVerificationStep 'browser' $flowNode @('node_modules/@playwright/test/cli.js','test') $flowWeb
            Invoke-FlowVerificationStep 'interface-lab' $flowNode @('node_modules/@playwright/test/cli.js','test','--config','playwright.lab.config.ts') $flowWeb
        } finally { $env:CI = $flowPreviousCI }
    }
    $flowNativeReceipt = Join-Path $flowVerifyRoot '_deps/windows-native-amd64/.flow-native-receipt.json'
    Copy-Item -LiteralPath $flowNativeReceipt -Destination (Join-Path $flowVerifyOutput 'native-toolchain.json')
    $flowVerifyReceipt.native_receipt_sha256 = (Get-FileHash -LiteralPath $flowNativeReceipt -Algorithm SHA256).Hash.ToLowerInvariant()
    $flowVerifyReceipt.passed = $true
} finally {
    $flowVerifyReceipt.finished_at = [DateTime]::UtcNow.ToString('o')
    $flowVerifyReceipt | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath (Join-Path $flowVerifyOutput 'receipt.json') -Encoding utf8
    Write-Host "Verification receipt: $flowVerifyOutput/receipt.json"
}
