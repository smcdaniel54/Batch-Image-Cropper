# batch-image-cropper — specification

## 1. Purpose

CLI tool: given flatbed-style scans (near-white background, darker photos), write **one rectified JPEG per detected photo**, a **QA overlay JPEG per successful source**, **`manifest.json`**, and **`quality_report.md`**. Optionally moves each completed source scan into **`processed/`** under the input directory (directory mode) or next to a single `-input` file (see §6)—not under `-out-dir`.

**In scope:** batch or single-file runs, top-level files only under an input directory (no subfolder recursion).

**Out of scope:** Interactive UI, cloud storage, formats beyond `.jpg` / `.jpeg` / `.png` as decoded by the Go standard library.

---

## 2. Constraints

| Item | Detail |
|------|--------|
| Runtime | Go, **no cgo** |
| Geometry | Corners and overlays use **source image pixel coordinates** (origin at image bounds minimum). |

---

## 3. CLI flags

| Flag | Default | Meaning |
|------|---------|---------|
| `-input` | *(empty)* | One image path. **Mutually exclusive** with `-input-dir`. |
| `-input-dir` | *(empty)* | Directory of images (files only, not subfolders). |
| *(neither)* | — | Same as `-input-dir ./input`. |
| `-out-dir` | `./output` | **Only** directory the tool creates for writes (see §4). |
| `-threshold` | `245` | Pixels strictly darker than this value are **foreground**; others are background. |
| `-min-area` | `20000` | Minimum 4-connected foreground area (px²) to keep a component. |
| `-padding` | `0` | Expand fitted quad from center before warp (px, approximate). |
| `-aspect` | `0` | If positive, center-crop warped result to this width÷height ratio. |

**Errors:** `-input` and `-input-dir` together ⇒ error. No supported images after filtering ⇒ error. Deliverable JPEG names are allowlisted so the **original scan basename** is never written into `-out-dir`.

**Startup:** Prints absolute paths for resolved input (file or directory) and `-out-dir`.

### 3.1 Example invocations (same as README)

No arguments (process `./input` → `./output`):

```powershell
.\batch-image-cropper.exe
```

Single file (output defaults to `./output`):

```powershell
.\batch-image-cropper.exe -input ".\scan.jpg" -threshold 245 -min-area 20000 -padding 10 -aspect 1.5
```

Entire folder (top-level files only, not subfolders):

```powershell
.\batch-image-cropper.exe -input-dir ".\scans" -out-dir ".\cropped" -threshold 245 -min-area 20000 -padding 10 -aspect 1.5
```

---

## 4. Output directory contract (`-out-dir`)

Everything the tool writes on success lives **only** under `-out-dir`. There are **no** other write roots (no sibling `debug/`, no alternate trees).

| File | Name pattern | Role |
|------|----------------|------|
| Cropped JPEGs | `<stem>_001.jpg`, `<stem>_002.jpg`, … | One per saved crop (`stem` = source basename without extension). |
| QA JPEG | `<stem>_000_qa.jpg` | One per source that produced ≥1 saved crop: full scan + quads, corners, 1-based labels. **Primary visual QA.** |
| Manifest | `manifest.json` | One file per run listing **every** saved crop (one manifest entry per crop JPEG). |
| Quality report | `quality_report.md` | Written **exactly once** per successful run; batch summary from the same data as the manifest. |

**Sorting:** `_000_qa` lexically before `_001`, `_002`, … for the same stem.

**Not written:** Rejected full-page-like crops (pipeline + size checks); stderr may warn.

Higher-level detection behavior (thresholding, components, warp, optional split of merged regions) is implementation detail; see `README.md` for an overview.

---

## 5. `manifest.json`

Top level: `version` (int, currently `1`), `entries` (array). There are **no** JSON fields for debug output or paths outside `-out-dir`.

Each **entry** corresponds to **one saved crop**:

| Field | Meaning |
|-------|---------|
| `source` | Scan path after the run (may be under `processed/`). |
| `output` | Crop JPEG basename. |
| `qa_image` | QA JPEG basename; same value on every entry from the same source. |
| `corners` | Four `[x,y]` pairs in source space. |
| `output_size` | `{ "width", "height" }` of the saved crop. |
| `mode` | Detection path (e.g. `quad_hull`, `rotated_min_area_rect`, axis-aligned fallbacks). |
| `confidence` | Number from the detector. |

---

## 6. `processed/` behavior

If a source yields **at least one** saved crop, the **original scan file** is moved into `processed/`:

- **Directory mode:** `…/<input-dir>/processed/`
- **Single `-input`:** `…/<parent-of-file>/processed/`

Collision names: `stem_2.ext`, `stem_3.ext`, … Manifest `source` is updated to the new path. **No move** if the run fails or that source produced zero saved crops.

---

## 7. Batch summary (stdout)

After success, prints counts (sources, photos, QA files, moves), fallback tallies by mode, warning count, and absolute path to `manifest.json`.

---

## 8. Compatibility

Filenames and manifest fields in §§3–6 are the stable surface for tooling. Breaking changes should bump `manifest.version` and be noted in release material.
