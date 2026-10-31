# Dot-source this file to configure this terminal only. No machine PATH changes.
$ErrorActionPreference = 'Stop'
$flowRoot = Split-Path -Parent $PSScriptRoot
$flowMsys = Join-Path $flowRoot '.tools\msys64'
$flowDeps = Join-Path $flowRoot '_deps\windows-native-amd64'
if (!(Test-Path -LiteralPath (Join-Path $flowMsys 'ucrt64\bin\gcc.exe'))) {
    throw 'Install the local MSYS2 UCRT64 compiler first; see build/README.md.'
}
$flowGoVersion = ((Get-Content -LiteralPath (Join-Path $flowRoot 'server\go.mod') |
    Select-String '^go (\S+)$').Matches.Groups[1].Value)
$flowGoBin = Join-Path $flowRoot ".tools\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go$flowGoVersion.windows-amd64\bin"
if (!(Test-Path -LiteralPath (Join-Path $flowGoBin 'go.exe'))) {
    $flowGoBin = Split-Path -Parent (Get-Command go.exe -ErrorAction Stop).Source
}
$env:PATH = "$flowGoBin;$flowMsys\ucrt64\bin;$flowMsys\usr\bin;$env:PATH"
$env:MSYSTEM = 'UCRT64'
$env:CHERE_INVOKING = 'yes'
$env:MSYS2_PATH_TYPE = 'inherit'
$env:GOPATH = Join-Path $flowRoot '.tools\gopath'
$env:GOCACHE = Join-Path $flowRoot '.tools\buffering-go-cache'
# This checkout can be owned by the isolated build account. Trust only this
# exact project for this terminal and its children, without editing .gitconfig.
$flowGitConfigCount = if ($env:GIT_CONFIG_COUNT) { [int]$env:GIT_CONFIG_COUNT } else { 0 }
[Environment]::SetEnvironmentVariable("GIT_CONFIG_KEY_$flowGitConfigCount", 'safe.directory', 'Process')
[Environment]::SetEnvironmentVariable("GIT_CONFIG_VALUE_$flowGitConfigCount", $flowRoot.Replace('\', '/'), 'Process')
$env:GIT_CONFIG_COUNT = [string]($flowGitConfigCount + 1)
$env:CGO_ENABLED = '1'
$env:CC = Join-Path $flowMsys 'ucrt64\bin\gcc.exe'
$env:CXX = Join-Path $flowMsys 'ucrt64\bin\g++.exe'
$env:PKG_CONFIG = Join-Path $flowMsys 'ucrt64\bin\pkg-config.exe'
$env:PKG_CONFIG_PATH = (Join-Path $flowDeps 'lib\pkgconfig').Replace('\', '/')
$env:PKG_CONFIG_LIBDIR = $env:PKG_CONFIG_PATH
$env:CGO_CXXFLAGS = '-DTSL_HAVE_LT_INTERNALS'
$env:FLOW_REQUIRE_CACHE_EXTENSION = '1'
$flowPcFile = Join-Path $flowDeps 'lib\pkgconfig\libtorrent-rasterbar.pc'
$flowArchive = Join-Path $flowDeps 'lib\libtorrent-rasterbar.a'
if ((Test-Path -LiteralPath $flowPcFile) -and (Test-Path -LiteralPath $flowArchive)) {
    # CGo does not invalidate cached flags when only a pkg-config file or
    # static archive changes. Include both identities in the compiler flags.
    $flowPcStamp = (Get-FileHash -LiteralPath $flowPcFile -Algorithm SHA256).Hash.Substring(0, 12)
    $flowArchiveStamp = (Get-FileHash -LiteralPath $flowArchive -Algorithm SHA256).Hash.Substring(0, 12)
    $env:CGO_CFLAGS = "-DTS_PC_STAMP=$flowPcStamp$flowArchiveStamp"
}
$flowLibPath = (Join-Path $flowDeps 'lib').Replace('\', '/')
$env:CGO_LDFLAGS = "-L$flowLibPath -static -static-libgcc -static-libstdc++ -lws2_32 -liphlpapi -lmswsock -lsecur32 -lcrypt32 -lbcrypt -ladvapi32 -luser32 -lshell32 -lgdi32"
Write-Host 'Flow native build environment enabled for this terminal.'
