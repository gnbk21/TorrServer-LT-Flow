[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidatePattern('^\d+\.\d+\.\d+$')][string]$CandidateVersion,
    [string]$Output = ''
)
$ErrorActionPreference = 'Stop'
$flowCompareRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $flowCompareRoot
. "$PSScriptRoot/Enter-FlowBuild.ps1"
$flowCompareGo = (Get-Command go.exe -ErrorAction Stop).Source
$flowComparePinned = ((Get-Content server/go.mod | Select-String '^go (\S+)$').Matches.Groups[1].Value)
if ($CandidateVersion -eq $flowComparePinned) { throw 'Candidate must differ from the pinned Go version.' }
if (!$Output) { $Output = Join-Path $flowCompareRoot ('.tools/go-comparison-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmss')) }
[void](New-Item -ItemType Directory -Path $Output)
$flowCompareOutput = (Resolve-Path -LiteralPath $Output).Path
$flowComparePreviousToolchain = $env:GOTOOLCHAIN
$flowCompareResults = [Collections.Generic.List[object]]::new()
$flowCompareSamples = [Collections.Generic.List[object]]::new()
function Get-FlowComparisonSourceDigest {
    $flowCompareHash = [Security.Cryptography.SHA256]::Create()
    try {
        $flowCompareFiles = Get-ChildItem -LiteralPath (Join-Path $flowCompareRoot 'server') -Recurse -File |
            Where-Object { $_.Extension -in @('.go','.h','.hpp','.c','.cpp','.mod','.sum') } | Sort-Object FullName
        $flowCompareInput = ($flowCompareFiles | ForEach-Object { $_.FullName.Substring($flowCompareRoot.Length) + ':' + (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }) -join "`n"
        return [BitConverter]::ToString($flowCompareHash.ComputeHash([Text.Encoding]::UTF8.GetBytes($flowCompareInput))).Replace('-','').ToLowerInvariant()
    } finally { $flowCompareHash.Dispose() }
}
$flowCompareInitialDigest = Get-FlowComparisonSourceDigest
$flowCompareReport = [ordered]@{schema_version=1;source=(& git rev-parse HEAD).Trim();modified=[bool](& git status --porcelain);pinned=$flowComparePinned;candidate=$CandidateVersion;passed=$false;promote=$false;results=$flowCompareResults;samples=$flowCompareSamples;decision='Retain the pinned runtime. This comparison is separate from eight-platform qualification and phone/public-swarm acceptance.'}
$flowCompareReport.source_digest = $flowCompareInitialDigest
function Invoke-FlowGoComparison {
    param([string]$Version,[string]$Name,[string[]]$Arguments)
    $env:GOTOOLCHAIN = "go$Version"
    $flowCompareLog = Join-Path $flowCompareOutput "$Version-$Name.log"
    Push-Location (Join-Path $flowCompareRoot 'server')
    try { & $flowCompareGo @Arguments *> $flowCompareLog; $flowCompareExit=$LASTEXITCODE } finally { Pop-Location }
    $flowCompareResults.Add(@{version=$Version;name=$Name;exit_code=$flowCompareExit;log=[IO.Path]::GetFileName($flowCompareLog);source_digest=(Get-FlowComparisonSourceDigest)})
    if ($flowCompareExit -ne 0) { return $false }
    if ($Name -like 'benchmark-*') {
        $flowCompareBenchmark = ''
        foreach ($flowCompareLine in Get-Content -LiteralPath $flowCompareLog) {
            if ($flowCompareLine -match '^(Benchmark\S+)') { $flowCompareBenchmark = $matches[1] }
            # Dependency logs can split the name and result across lines.
            if ($flowCompareBenchmark -and $flowCompareLine -match '(?:^|\s)(\d+)\s+([0-9.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op') {
                $flowCompareSamples.Add(@{version=$Version;rotation=$Name;benchmark=$flowCompareBenchmark;iterations=[long]$matches[1];ns_per_op=[double]::Parse($matches[2],[Globalization.CultureInfo]::InvariantCulture);bytes_per_op=[long]$matches[3];allocs_per_op=[long]$matches[4]})
                $flowCompareBenchmark = ''
            }
        }
    }
    return $true
}
try {
    foreach ($flowCompareVersion in @($flowComparePinned,$CandidateVersion)) {
        if (!(Invoke-FlowGoComparison $flowCompareVersion 'version' @('version'))) { continue }
        [void](Invoke-FlowGoComparison $flowCompareVersion 'compatibility' @('test','-p','2','-json','-count=1','-timeout','180s','./flow','./diagnostics','./settings','./lt','./torr','./torr/storage/torrstor','./web/api'))
        [void](Invoke-FlowGoComparison $flowCompareVersion 'build' @('build','-pgo=off','-ldflags',"-X server/version.Version=MatriX.148.Flow-toolchain-$flowCompareVersion",'-o',(Join-Path $flowCompareOutput "Flow-go$flowCompareVersion.exe"),'./cmd'))
    }
    for ($flowCompareRotation=0;$flowCompareRotation -lt 3;$flowCompareRotation++) {
        $flowCompareOrder = if ($flowCompareRotation % 2) { @($CandidateVersion,$flowComparePinned) } else { @($flowComparePinned,$CandidateVersion) }
        foreach ($flowCompareVersion in $flowCompareOrder) {
            [void](Invoke-FlowGoComparison $flowCompareVersion "benchmark-$flowCompareRotation" @('test','-run','^$','-pgo=off','-bench','^BenchmarkLibrarySingleUpsert$','-benchtime','3x','-count=1','-benchmem','./settings'))
        }
    }
    $flowCompareReport.native_receipt_sha256 = (Get-FileHash -LiteralPath '_deps/windows-native-amd64/.flow-native-receipt.json' -Algorithm SHA256).Hash.ToLowerInvariant()
    $flowCompareReport.passed = !@($flowCompareResults | Where-Object exit_code -ne 0).Count -and $flowCompareSamples.Count -ge 6
} finally {
    $env:GOTOOLCHAIN = $flowComparePreviousToolchain
    $flowCompareReport.source_changed = (Get-FlowComparisonSourceDigest) -ne $flowCompareInitialDigest -or @($flowCompareResults | Where-Object source_digest -ne $flowCompareInitialDigest).Count -gt 0
    if ($flowCompareReport.source_changed) { $flowCompareReport.passed = $false; $flowCompareReport.decision = 'Source changed during comparison. Retain all observations, do not qualify timings, and rerun on a fixed source.' }
    $flowCompareReport | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath (Join-Path $flowCompareOutput 'comparison.json') -Encoding utf8
    Write-Host "Toolchain comparison: $flowCompareOutput/comparison.json"
}
if (!$flowCompareReport.passed) { throw 'Toolchain comparison incomplete or failed; individual results retained and the pinned runtime remains unchanged.' }
