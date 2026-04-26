# Development only: move files from <InputDir>/processed back into <InputDir>/; optionally remove ./output.
# Does not run the cropper. ASCII-only console messages.

param(
    [string]$InputDir = ".\input",
    [switch]$RemoveOutput
)

$ErrorActionPreference = "Stop"

$ProcessedDir = Join-Path $InputDir "processed"
$OutputDir = ".\output"

Write-Host "Dev reset starting..."
Write-Host "Input:     $InputDir"
Write-Host "Processed: $ProcessedDir"

if (!(Test-Path -Path $InputDir -PathType Container)) {
    New-Item -ItemType Directory -Path $InputDir | Out-Null
    Write-Host "Created input dir: $InputDir"
}

if (Test-Path -Path $ProcessedDir -PathType Container) {
    $files = Get-ChildItem -Path $ProcessedDir -File
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
    if ($files.Count -eq 0) {
        Write-Host "Processed dir empty (no files to move)."
    }
}
else {
    Write-Host "No processed dir found."
}

if ($RemoveOutput -and (Test-Path -Path $OutputDir -PathType Container)) {
    Remove-Item -Path $OutputDir -Recurse -Force
    Write-Host "Removed output dir: $OutputDir"
}

Write-Host "Dev reset complete."
