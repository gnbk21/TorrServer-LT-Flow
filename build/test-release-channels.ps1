# Offline channel-selection boundary regression; no executable or network use.
$ErrorActionPreference = 'Stop'
. (Join-Path (Split-Path $PSScriptRoot -Parent) 'distribution/FlowRelease.ps1')
$fixtureCommit = 'a' * 40
function Release([string]$Tag, [bool]$Preview, [bool]$Draft = $false) {
    [pscustomobject]@{tag_name=$Tag;prerelease=$Preview;draft=$Draft;assets=@([pscustomobject]@{name='release.json';browser_download_url="https://github.com/gnbk21/TorrServer-LT-Flow/releases/download/$Tag/release.json"})}
}
$fixtureReleases = @(
    (Release 'MatriX.145.Flow-dev-unknown' $false),
    (Release 'MatriX.145.Flow-preview.999' $false),
    (Release 'MatriX.145.Flow-v1.0.0' $true),
    (Release 'MatriX.145.Flow-v99.0.0' $false $true),
    (Release 'MatriX.145.Flow-preview.2' $true),
    (Release 'MatriX.145.Flow-v0.1.0' $false)
)
function Invoke-RestMethod {
    param([string]$Uri, $Headers, $TimeoutSec)
    if ($Uri -like 'https://api.github.com/*') {
        # Match Invoke-RestMethod's JSON-array output rather than PowerShell
        # function return enumeration, which hid the live API regression.
        Write-Output -NoEnumerate $fixtureReleases
        return
    }
    $tag = ($Uri -split '/')[-2]
    $channel = 'stable'
    if ($tag -match 'preview|alpha|beta|rc') { $channel = 'preview' }
    [pscustomobject]@{schema_version=1;repository='gnbk21/TorrServer-LT-Flow';channel=$channel;Version=$tag;commit=$fixtureCommit}
}
if ((Get-FlowRelease stable).Version -ne 'MatriX.145.Flow-v0.1.0') { throw 'Stable selection accepted a draft, preview or development tag.' }
if ((Get-FlowRelease preview).Version -ne 'MatriX.145.Flow-preview.2') { throw 'Preview selection accepted a stable tag.' }
$fixtureReleases = @((Release 'MatriX.145.Flow-v1.0.0-rc.1' $true))
if ((Get-FlowRelease preview).Version -ne 'MatriX.145.Flow-v1.0.0-rc.1') { throw 'Versioned release candidate was rejected.' }
$fixtureReleases = @((Release 'MatriX.145.Flow-dev-unknown' $true))
$rejected = $false
try { [void](Get-FlowRelease preview) } catch { $rejected = $_.Exception.Message -like 'No published Flow*' }
if (-not $rejected) { throw 'Development-only releases were accepted as previews.' }
Write-Output 'Release channel selection passed.'
