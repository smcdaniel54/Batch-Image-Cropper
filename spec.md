# batch-image-cropper — product specification

## 1. Purpose

**batch-image-cropper** is a command-line tool that takes one or more flatbed-style scan images (near-white background, darker photos) and produces **one rectified JPEG per detected photo**, plus **machine-readable metadata** and a **single QA overlay per source scan** for visual inspection.

**In scope:** batch or single-file processing, deterministic naming, manifest and quality summary, optional post-move of sources into `processed/`.

**Out of scope:** subfolder recursion for inputs, RAW or TIFF beyond what Go’s image decoders support, interactive UI, cloud or database integration.

---

## 2. Technical constraints

| Constraint | Detail |
|------------|--------|
| Language | Go, **no cgo** |
| Inputs | `.jpg`, `.jpeg`, `.png` (decoded via standard library + `_` imports) |
| Photo outputs | JPEG only, allowlisted names (see §4) |
| Coordinate space | Detection geometry is expressed in **source pixel coordinates** (origin at source image bounds minimum). |

---

## 3. CLI contract

### 3.1 Flags

| Flag | Default | Semantics |
|------|---------|-----------|
| `-input` | *(empty)* | Absolute or relative path to **one** image. Mutually exclusive with `-input-dir`. |
| `-input-dir` | *(empty)* | Directory of **top-level** image files only (no recursive walk). |
| *(both omitted)* | — | Equivalent to `-input-dir` `./input`. |
| `-out-dir` | `./output` | Directory created if missing; all artifacts written here. |
| `-threshold` | `245` | Luminance (0–255): pixels **strictly below** this value are **foreground**; at or above are **background**. |
| `-min-area` | `20000` | Minimum 4-connected foreground area (pixels²) for a component to be considered a photo candidate. |
| `-padding` | `0` | Expands the fitted quad from its center before warp (pixels, approximate). |
| `-aspect` | `0` | If **> 0**, after warp the crop is **center-cropped** so width ÷ height equals this value. |

**Errors:** Using `-input` and `-input-dir` together fails. No supported images after filtering yields an error. Output filenames are validated so the original scan basename is never written as a deliverable JPEG.

### 3.2 Startup logging

The process prints resolved **absolute** paths for the effective input (file or directory) and output directory before processing.

---

## 4. Outputs (authoritative list)

All paths are relative to `-out-dir` unless noted.

| Artifact | Pattern | When written |
|----------|---------|----------------|
| Cropped photos | `<stem>_001.jpg`, `<stem>_002.jpg`, … | One per accepted crop from that source (`stem` = source basename without extension). |
| QA overlay | `<stem>_000_qa.jpg` | Once per source that produced **at least one** saved crop. Full-scan overlay with quads, corners, and 1-based indices. |
| Manifest | `manifest.json` | Always after a successful run that created the output dir. |
| Quality report | `quality_report.md` | Always with the manifest; summarizes the same batch. |

**Sorting:** For a given stem, `_000_qa` sorts before `_001`, `_002`, … by design.

**Rejected crops:** Full-page-like crops (guards in pipeline and post-check vs source size) are not written; a warning may be logged.

---

## 5. Processing pipeline (normative overview)

1. **Decode** source to RGBA.
2. **Binarize** using luminance vs `-threshold` (foreground = darker).
3. **Label** foreground with **4-connectivity**; build **axis-aligned bounding regions** filtered by `-min-area`.
4. **Sort** regions top-to-bottom, then left-to-right.
5. **Full-page filter:** Drop regions that cover the scan in a sheet-like way (bbox / side / quad heuristics).
6. **Optional split:** Inside a region bbox, if the binary mask shows a strong **vertical or horizontal** background “separator band” (fixed heuristics: band width 3–10 px, ≥90% background along span, away from bbox edges, both halves still ≥ `-min-area` foreground for that label), treat as **two** clip regions and run extraction **twice** (reading order: left then right, or top then bottom).
7. **Per region (or split half):** Sample **border** foreground pixels, build **convex hull**, derive a **quad** (hull quadrilateral or minimum-area rectangle), validate, apply **padding**, **homography warp** to a rectangle, then optional **aspect** center-crop and **margin trim** on the warped result.
8. **Emit** JPEGs, build manifest entries, render QA overlay from saved crop geometry.

Modes recorded per crop include at least: `quad_hull`, `rotated_min_area_rect`, `axis_aligned`, `axis_aligned_invalid_quad`, `axis_aligned_homography_fail`.

---

## 6. Manifest (`manifest.json`)

- **Version:** integer `version` (currently `1`).
- **Entries:** one object per **saved** crop (not per skipped or rejected candidate).

Per entry:

| Field | Type | Meaning |
|-------|------|---------|
| `source` | string | Path to scan at end of run (may point under `processed/` after move). |
| `output` | string | Basename of the crop JPEG. |
| `qa_image` | string | Basename of `<stem>_000_qa.jpg` for that source (repeated on each entry from the same source). |
| `corners` | 4×2 float array | Quad corners in source pixel space (TL–TR–BR–BL ordering as produced by the detector). |
| `output_size` | `{ width, height }` | Saved crop dimensions. |
| `mode` | string | Detection / fallback path used. |
| `confidence` | number | Scalar confidence from the detector. |

---

## 7. Post-processing: `processed/`

After a source yields **at least one** saved crop, the **original file** is moved under `processed/`:

- **Directory input:** `processed/` is under that input directory.
- **Single `-input`:** `processed/` is next to the input file’s directory.

Name collisions in `processed/` use `stem_2.ext`, `stem_3.ext`, … Manifest `source` paths are updated to the new location. **No move** occurs if the run fails or if zero crops were saved for that file.

---

## 8. Batch summary (stdout)

After a successful run, a short summary includes at least: source image count, photos extracted, QA images written, files moved, fallback counts by mode, warning count, and absolute path to `manifest.json`.

---

## 9. Development helper (non-normative)

`scripts/dev-reset-input.ps1` resets a local `input` / `processed` / `output` dev loop and invokes the built executable; behavior is described in `README.md` and may change independently of this spec.

---

## 10. Change policy

Behavior and filenames in §§3–8 are **stable API** for downstream tooling. Changes to detection heuristics (§5) or manifest schema should bump `version` and be documented in release notes when they affect compatibility.
