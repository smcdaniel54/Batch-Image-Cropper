# Development: reset input/processed + output, build exe if missing, then run batch-image-cropper.
# Run from anywhere; working directory is set to the repo root (next to go.mod).
# ASCII-only console messages.

param(
    [string]$InputDir = ".\input",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$CropperArgs
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $RepoRoot

if ($null -eq $CropperArgs -or $CropperArgs.Count -eq 0) {
    # Explicit -input-dir so dev runs match batch mode; -debug for QA overlays.
    $CropperArgs = @("-input-dir", $InputDir, "-debug")
}

Write-Host "--- reset ---"
Write-Host "Repo:      $RepoRoot"
Write-Host "Input:     $InputDir"
$ProcessedDir = Join-Path $InputDir "processed"
$OutputDir = ".\output"
Write-Host "Processed: $ProcessedDir"
Write-Host "Output:    $OutputDir"

if (!(Test-Path -Path $InputDir -PathType Container)) {
    New-Item -ItemType Directory -Path $InputDir | Out-Null
    Write-Host "Created input dir: $InputDir"
}

if (Test-Path -Path $ProcessedDir -PathType Container) {
    $files = Get-ChildItem -Path $ProcessedDir -File

    if ($files.Count -eq 0) {
        Write-Host "Processed dir empty (no files to move)."
    }

    foreach ($file in $files) {
        $dest = Join-Path $InputDir $file.Name

        if (Test-Path -Path $dest) {
            $stem = [System.IO.Path]::GetFileNameWithoutExtension($file.Name)
            $ext = [System.IO.Path]::GetExtension($file.Name)
            $i = 2

            do {
                $candidate = Join-Path $InputDir ("{0}_{1}{2}" -f $stem, $i, $ext)
                $i++
            } while (Test-Path -Path $candidate)

            $dest = $candidate
        }

        Move-Item -Path $file.FullName -Destination $dest
        Write-Host "moved: $($file.FullName) -> $dest"
    }
}
else {
    Write-Host "No processed dir found."
}

if (Test-Path -Path $OutputDir -PathType Container) {
    Remove-Item -Path $OutputDir -Recurse -Force
    Write-Host "Removed output dir: $OutputDir"
}
else {
    Write-Host "No output dir to remove."
}

Write-Host "--- build ---"
$exePath = Join-Path $RepoRoot "batch-image-cropper.exe"
if (Test-Path -LiteralPath $exePath) {
    Write-Host "Found batch-image-cropper.exe"
}
else {
    Write-Host "Building batch-image-cropper.exe..."
    & go build -o batch-image-cropper.exe .
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

Write-Host "--- run ---"
$exeRun = Join-Path $RepoRoot "batch-image-cropper.exe"
Write-Host "Command: $exeRun $($CropperArgs -join ' ')"
& $exeRun @CropperArgs
exit $LASTEXITCODE
