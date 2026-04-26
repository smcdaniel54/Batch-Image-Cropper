# batch-image-cropper

Pure Go (no cgo) CLI for splitting flatbed or sheet-fed scans: **detect each photo on a near-white background**, run a **single-step perspective (homography) warp** to straighten, optionally fit an **aspect ratio**, and write one JPEG per photo plus **`manifest.json`** and **`quality_report.md`** (see `spec.md` for the output contract).

## Requirements

- [Go 1.23+](https://go.dev/dl/) (module `go 1.23.0`; tested on Windows 11; works on any Go-supported OS)

## Build on Windows 11

Open **PowerShell** or **cmd** in the project folder (where `go.mod` lives):

```powershell
go build -o batch-image-cropper.exe .
```

This produces `batch-image-cropper.exe` in the current directory.

## Usage

**Defaults:** if you omit both `-input` and `-input-dir`, that is the same as `-input-dir ./input`. If you omit `-out-dir`, deliverables go to `./output` (created if needed). All JPEG crops, QA overlays, `manifest.json`, and `quality_report.md` are written **only** under `-out-dir`. The program prints the resolved **absolute** input and output paths on startup. You cannot use `-input` and `-input-dir` together.

### No arguments (process `./input` → `./output`)

```powershell
.\batch-image-cropper.exe
```

### Single file (output defaults to `./output`)

```powershell
.\batch-image-cropper.exe -input ".\scan.jpg" -threshold 245 -min-area 20000 -padding 10 -aspect 1.5
```

### Entire folder (top-level files only, not subfolders)

```powershell
.\batch-image-cropper.exe -input-dir ".\scans" -out-dir ".\cropped" -threshold 245 -min-area 20000 -padding 10 -aspect 1.5
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | (none) | One image path. **Mutually exclusive** with `-input-dir`. |
| `-input-dir` | (none) | Directory of images (files only, not subfolders). If both `-input` and `-input-dir` are omitted, defaults to `./input`. |
| `-out-dir` | `./output` | **Only** directory the tool uses for deliverables—JPEGs, `manifest.json`, and `quality_report.md` (see `spec.md` §4); created if missing. |
| `-threshold` | `245` | Pixels strictly darker than this value are **foreground**; others are background. |
| `-min-area` | `20000` | Minimum 4-connected foreground area (px²) to keep a component. |
| `-padding` | `0` | Expand fitted quad from center before warp (px, approximate). |
| `-aspect` | `0` | If positive, center-crop warped result to this width÷height ratio. |

Supported inputs: **`.jpg`**, **`.jpeg`**, **`.png`**.

## Outputs

All of the following live **only** under `-out-dir` (no debug directory and no deliverables written elsewhere):

- **QA overlay** (per scan that produced at least one crop): `<source-stem>_000_qa.jpg` — full scan with each crop’s quad, corners, and index labels. This is the **primary visual inspection** artifact for detection quality.
- **Cropped images:** `<source-stem>_001.jpg`, `<source-stem>_002.jpg`, …
- **`manifest.json`:** one entry per **saved** crop (`qa_image` is the same basename on every entry from a source), plus `source`, `output`, `corners`, `output_size`, `mode`, and `confidence`. Modes include `quad_hull`, `rotated_min_area_rect`, `axis_aligned`, `axis_aligned_invalid_quad` (quad failed validation), and `axis_aligned_homography_fail` (matrix solve/invert failed).
- **`quality_report.md`:** written **once** per successful run, batch summary from the same data as the manifest.
- **Processed scans:** after each source file yields at least one photo, the original scan is moved to `processed/` under the input directory (when using a folder or default `./input`), or to `processed/` next to a single input file. Collisions are resolved with `name_2.ext`, `name_3.ext`, etc. The manifest `source` path is updated to the new location. Nothing is moved if the run fails, or if a scan produced zero photos.

## Development helper (PowerShell)

This is the **normal dev quality loop**: move **files** (not subfolders) from `input/processed/` back into `input/`, **always** delete `./output` if it exists, then run **`batch-image-cropper.exe`** (building it with **`go build`** first if the exe is missing). Name clashes in `input/` are resolved as `name_2.ext`, `name_3.ext`, etc.

When you pass no extra arguments, the script runs **`batch-image-cropper.exe -input-dir <InputDir>`** (same as batch mode on your input folder; QA JPEGs are written whenever crops are saved). Pass any extra cropper flags after the script parameters; they replace that default and are forwarded as-is. The script **`Set-Location`** to the repo root (parent of `scripts/`), so you can run it from any directory.

```powershell
.\scripts\dev-reset-input.ps1
```

Custom input folder and extra flags (unbound arguments are passed through to `batch-image-cropper.exe`):

```powershell
.\scripts\dev-reset-input.ps1 -InputDir ".\my-input" -threshold 240 -min-area 15000
```

## Algorithm notes

Foreground is found by **luminance thresholding**, then **4-connected** components, sorted **top-to-bottom, then left-to-right**. For each region, border samples feed a **convex hull**; either a 4-vertex hull or a **minimum-area rectangle** (angular sweep) supplies four corners. Corners are ordered by **centroid + polar angle**, then normalized to TL–TR–BR–BL with winding matched to the destination rectangle. Quads are **validated** (minimum area, edge length, aspect ratio, self-intersection) before warping; failed candidates fall back to the minimum-area rectangle or an axis-aligned crop, with matching `manifest.json` modes and confidence.

## License

This project is provided as-is; add a license as needed.
