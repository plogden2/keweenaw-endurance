param(
    [Parameter(Mandatory = $true)]
    [string]$RunDir
)

$ErrorActionPreference = 'Stop'

$reader = Join-Path $RunDir 'reader.webm'
$laptop = Join-Path $RunDir 'spectator-laptop.webm'
$iphone = Join-Path $RunDir 'spectator-iphone.webm'
$out = Join-Path $RunDir 'side-by-side-1440p.mp4'

$inputs = @($reader, $laptop, $iphone)
foreach ($path in $inputs) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Missing input video: $path"
    }
}

function Get-LabelDrawText {
    param([string]$Text)

    $filters = & ffmpeg -hide_banner -filters 2>&1
    if ($LASTEXITCODE -ne 0 -or ($filters -notmatch '\bdrawtext\b')) {
        return ''
    }

    $font = @(
        (Join-Path $env:WINDIR 'Fonts\arial.ttf'),
        (Join-Path $env:WINDIR 'Fonts\segoeui.ttf')
    ) | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1

    if (-not $font) {
        return ''
    }

    $fontPath = ($font -replace '\\', '/') -replace ':', '\:'
    $escapedText = $Text -replace "'", "''"

    return ",drawtext=fontfile='$fontPath':text='$escapedText':fontcolor=white:fontsize=48:box=1:boxcolor=black@0.5:boxborderw=10:x=20:y=20"
}

$labelReader = Get-LabelDrawText 'Reader'
$labelLaptop = Get-LabelDrawText 'Laptop'
$labelIphone = Get-LabelDrawText 'iPhone'

$filterComplex = @(
    "[0:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1$labelReader[v0]"
    "[1:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1$labelLaptop[v1]"
    "[2:v]scale=853:1440:force_original_aspect_ratio=decrease,pad=853:1440:(ow-iw)/2:(oh-ih)/2,setsar=1$labelIphone[v2]"
    '[v0][v1][v2]hstack=inputs=3[v]'
) -join ';'

ffmpeg -y `
    -i $reader -i $laptop -i $iphone `
    -filter_complex $filterComplex `
    -map '[v]' -an -c:v libx264 -crf 20 -pix_fmt yuv420p $out

if ($LASTEXITCODE -ne 0) {
    throw "ffmpeg failed with exit code $LASTEXITCODE"
}

Write-Host "[compose-bluffet-hardware-video] wrote $out"
