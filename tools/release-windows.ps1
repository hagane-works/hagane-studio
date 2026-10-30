[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._-]*$')]
    [string]$Version,
    [switch]$SkipFrontend,
    [switch]$SkipChecks,
    [switch]$SkipSourceArchive
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$releaseRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'dist\release'))
$versionRoot = [IO.Path]::GetFullPath((Join-Path $releaseRoot $Version))
$releasePrefix = $releaseRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $versionRoot.StartsWith($releasePrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Release output escaped dist\release.'
}

function Invoke-Checked {
    param([string]$Program, [string[]]$Arguments, [string]$WorkingDirectory = $repoRoot)
    Push-Location $WorkingDirectory
    try {
        & $Program @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$Program failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }
}

function Copy-ReleaseFile {
    param([string]$Source, [string]$Destination)
    if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) {
        throw "Required release file is missing: $Source"
    }
    Copy-Item -LiteralPath $Source -Destination $Destination -Force
}

$licensePath = Join-Path $repoRoot 'LICENSE'
if (-not (Test-Path -LiteralPath $licensePath) -or -not (Select-String -LiteralPath $licensePath -SimpleMatch 'GNU GENERAL PUBLIC LICENSE' -Quiet)) {
    throw 'A complete GNU GPL license is required before building a public release.'
}
$cwtoolsRoot = Join-Path $repoRoot 'third_party\cwtools-hoi4-config'
if (-not (Test-Path -LiteralPath (Join-Path $cwtoolsRoot 'Config') -PathType Container)) {
    throw 'CWTools configuration is missing. Initialize the git submodule first.'
}
if (-not $SkipFrontend) {
    Invoke-Checked 'npm' @('ci') (Join-Path $repoRoot 'web')
    Invoke-Checked 'npm' @('run', 'build') (Join-Path $repoRoot 'web')
}
if (-not (Test-Path -LiteralPath (Join-Path $repoRoot 'web\dist\index.html') -PathType Leaf)) {
    throw 'web\dist is missing. Build the frontend before packaging.'
}

if (-not $SkipChecks) {
    Invoke-Checked 'go' @('test', '-vet=off', './...')
    Invoke-Checked 'npm' @('run', 'typecheck') (Join-Path $repoRoot 'web')
    Invoke-Checked 'npm' @('test', '--', '--run') (Join-Path $repoRoot 'web')
}

if (Test-Path -LiteralPath $versionRoot) {
    Remove-Item -LiteralPath $versionRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $versionRoot -Force | Out-Null

$architectures = @(
    [pscustomobject]@{ GoArch = '386'; Name = 'x86' },
    [pscustomobject]@{ GoArch = 'amd64'; Name = 'x64' },
    [pscustomobject]@{ GoArch = 'arm64'; Name = 'arm64' }
)

$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$oldCgo = $env:CGO_ENABLED
try {
    $env:GOOS = 'windows'
    $env:CGO_ENABLED = '0'
    foreach ($architecture in $architectures) {
        $env:GOARCH = $architecture.GoArch
        $packageName = "HaganeWorks-$Version-windows-$($architecture.Name)"
        $packageDir = Join-Path $versionRoot $packageName
        New-Item -ItemType Directory -Path $packageDir -Force | Out-Null

        Invoke-Checked 'go' @('build', '-trimpath', '-buildvcs=false', '-ldflags', "-s -w -H=windowsgui -X main.version=$Version", '-o', (Join-Path $packageDir 'HaganeWorks.exe'), './cmd/haganeworks')
        Invoke-Checked 'go' @('build', '-trimpath', '-buildvcs=false', '-ldflags', '-s -w', '-o', (Join-Path $packageDir 'hw.exe'), './cmd/hw')

        Copy-ReleaseFile (Join-Path $repoRoot 'docs\release\START-HERE.zh-CN.txt') (Join-Path $packageDir 'START-HERE.zh-CN.txt')
        Copy-ReleaseFile (Join-Path $repoRoot 'docs\release\START-HERE.en.txt') (Join-Path $packageDir 'START-HERE.en.txt')
        Copy-ReleaseFile (Join-Path $repoRoot 'LICENSE') (Join-Path $packageDir 'LICENSE')
        Copy-ReleaseFile (Join-Path $repoRoot 'THIRD_PARTY_NOTICES.md') (Join-Path $packageDir 'THIRD_PARTY_NOTICES.md')
		Set-Content -LiteralPath (Join-Path $packageDir 'VERSION.txt') -Value "Hagane Works $Version`nWindows $($architecture.Name)" -Encoding utf8
		Set-Content -LiteralPath (Join-Path $packageDir 'SOURCE-CODE.txt') -Value "The corresponding source is published beside this archive as HaganeWorks-$Version-source.zip on the same release page. Hagane Works is licensed under GNU GPL v3.0 only. A distributor of this binary must keep the corresponding source available under the GPL." -Encoding utf8

        New-Item -ItemType Directory -Path (Join-Path $packageDir 'docs') -Force | Out-Null
        Copy-ReleaseFile (Join-Path $repoRoot 'docs\cli.md') (Join-Path $packageDir 'docs\cli.md')

        $cwtoolsDest = Join-Path $packageDir 'third_party\cwtools-hoi4-config'
        New-Item -ItemType Directory -Path $cwtoolsDest -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $cwtoolsRoot 'Config') -Destination (Join-Path $cwtoolsDest 'Config') -Recurse -Force
        Copy-ReleaseFile (Join-Path $cwtoolsRoot 'LICENSE') (Join-Path $cwtoolsDest 'LICENSE')
        if (Test-Path -LiteralPath (Join-Path $cwtoolsRoot 'README.md')) {
            Copy-ReleaseFile (Join-Path $cwtoolsRoot 'README.md') (Join-Path $cwtoolsDest 'README.md')
        }

        $zipPath = Join-Path $versionRoot "$packageName.zip"
        Compress-Archive -Path (Join-Path $packageDir '*') -DestinationPath $zipPath -CompressionLevel Optimal
        Remove-Item -LiteralPath $packageDir -Recurse -Force
    }
}
finally {
    $env:GOOS = $oldGoos
    $env:GOARCH = $oldGoarch
    $env:CGO_ENABLED = $oldCgo
}

if (-not $SkipSourceArchive) {
    $tracked = @(& git -C $repoRoot ls-files)
    if ($LASTEXITCODE -ne 0 -or $tracked.Count -eq 0) {
        throw 'The source archive requires a committed git tree.'
    }
    $sourceZip = Join-Path $versionRoot "HaganeWorks-$Version-source.zip"
    Invoke-Checked 'git' @('archive', '--format=zip', "--prefix=HaganeWorks-$Version-source/", "--output=$sourceZip", 'HEAD')
}

$checksumLines = Get-ChildItem $versionRoot -File -Filter '*.zip' | Sort-Object Name | ForEach-Object {
    $hash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($_.Name)"
}
Set-Content -LiteralPath (Join-Path $versionRoot 'SHA256SUMS.txt') -Value $checksumLines -Encoding utf8
Get-ChildItem $versionRoot -File | Sort-Object Name | Select-Object Name, Length
