param(
    [string]$OutFile
)

$ErrorActionPreference = "Stop"

$rootDir = Split-Path -Parent $PSScriptRoot
if (-not $OutFile) {
    $OutFile = Join-Path $rootDir "test\test.srt"
}

Set-Location $rootDir

go run . transcript `
    --file test/test.mp3 `
    --output $OutFile `
    --source-lang zh `
    --target-lang en

if ($LASTEXITCODE -ne 0) {
    throw "transcript command failed with exit code $LASTEXITCODE"
}

Write-Host ""
Write-Host "Preview:"
Get-Content $OutFile | Select-Object -First 40
