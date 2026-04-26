// Package main is the batch-image-cropper CLI: extract photos from flatbed scans in pure Go.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"photo-cropper/internal/cropper"
	"photo-cropper/internal/fsutil"
	"photo-cropper/internal/manifest"
	"photo-cropper/internal/moveprocessed"
	"photo-cropper/internal/qualityreport"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		input     = flag.String("input", "", "single input image (optional; .jpg, .jpeg, .png)")
		inputDir  = flag.String("input-dir", "", "folder of scan images (optional; default ./input if neither -input nor -input-dir)")
		outDir    = flag.String("out-dir", "", "output directory (default: ./output)")
		threshold = flag.Int("threshold", 245, "pixels darker than this (0-255) are treated as foreground (near-white background)")
		minArea   = flag.Int("min-area", 20000, "minimum area in pixels for a component")
		padding   = flag.Int("padding", 0, "expand each detected region by this many pixels (from center)")
		aspect    = flag.Float64("aspect", 0, "if >0, center-crop to this width/height after warp")
		debug     = flag.Bool("debug", false, "write debug overlays to <out-dir>/debug")
	)
	flag.Parse()
	singleInput := *input != ""

	if *input != "" && *inputDir != "" {
		return fmt.Errorf("cannot use -input and -input-dir together")
	}

	// 1) Resolve input and collect at least zero valid image paths (no output dirs created yet).
	var warnCount int
	warnLog := func(msg string) {
		warnCount++
		fmt.Fprintln(os.Stderr, msg)
	}
	inputs, inputAbs, err := gatherImageInputs(*input, *inputDir, warnLog)
	if err != nil {
		return err
	}

	// 2) Resolve output path for logging only; still no directories created.
	out := *outDir
	if out == "" {
		out = "./output"
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "input: ", inputAbs)
	fmt.Fprintln(os.Stdout, "output:", outAbs)

	if len(inputs) == 0 {
		return fmt.Errorf("no supported image files to process")
	}

	// 3) At least one file to process: create output and optional debug layout.
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return fmt.Errorf("create out-dir: %w", err)
	}
	debugDir := ""
	if *debug {
		debugDir = filepath.Join(outAbs, "debug")
		if err := os.MkdirAll(debugDir, 0o755); err != nil {
			return fmt.Errorf("create debug dir: %w", err)
		}
	}

	var entries []manifest.Entry
	fallbackByMode := map[string]int{
		"axis_aligned":                 0,
		"axis_aligned_invalid_quad":    0,
		"axis_aligned_homography_fail": 0,
	}
	opts := cropper.Options{
		Threshold: *threshold,
		MinArea:   *minArea,
		Padding:   *padding,
		Aspect:    *aspect,
		DebugDir:  debugDir,
	}

	sourceCount := 0
	filesMoved := 0
	qaImagesWritten := 0
	for _, path := range inputs {
		source, imgs, metas, perr := cropper.ProcessScan(path, opts)
		if perr != nil {
			return perr
		}
		sourceCount++
		var qaName string
		if len(imgs) > 0 {
			var nerr error
			qaName, nerr = outputQAName(path)
			if nerr != nil {
				return nerr
			}
		}
		savedN := 0
		var qaMetas []cropper.Meta
		for i := range imgs {
			if cropLooksFullPageAgainstSource(imgs[i], source) {
				warnLog("rejected full-page crop candidate")
				continue
			}
			savedN++
			outName, err := outputPhotoName(path, savedN)
			if err != nil {
				return err
			}
			if strings.EqualFold(outName, filepath.Base(path)) {
				return fmt.Errorf("internal: output name must not match input: %q", outName)
			}
			outPath := filepath.Join(outAbs, outName)
			if err := saveJPEG(outPath, imgs[i], 95); err != nil {
				return err
			}
			bd := imgs[i].Bounds()
			mode := metas[i].Mode
			qaMetas = append(qaMetas, metas[i])
			entries = append(entries, manifest.Entry{
				Source:     path,
				Output:     outName,
				QaImage:    qaName,
				Corners:    metaCornersToSlice(metas[i]),
				OutputSize: manifest.Size{Width: bd.Dx(), Height: bd.Dy()},
				Mode:       mode,
				Confidence: metas[i].Confidence,
			})
			if _, ok := fallbackByMode[mode]; ok {
				fallbackByMode[mode]++
			}
		}
		if savedN > 0 && source != nil {
			qaPath := filepath.Join(outAbs, qaName)
			qa := cropper.QaScanOverlay(source, qaMetas)
			if err := saveQAJPEG(qaPath, qa, 90); err != nil {
				return err
			}
			qaImagesWritten++
		}
		if moveprocessed.MoveAfterExtraction(savedN) {
			base := inputAbs
			if singleInput {
				base = filepath.Dir(path)
			}
			dst, merr := moveprocessed.MoveToProcessed(path, base)
			if merr != nil {
				return fmt.Errorf("move processed: %w", merr)
			}
			fmt.Fprintf(os.Stdout, "moved: %s -> %s\n", path, dst)
			for i := range entries {
				if entries[i].Source == path {
					entries[i].Source = dst
				}
			}
			filesMoved++
		}
	}

	manifestPath := filepath.Join(outAbs, "manifest.json")
	if err := writeManifest(manifestPath, entries); err != nil {
		return err
	}
	qualityPath := filepath.Join(outAbs, "quality_report.md")
	if err := qualityreport.WriteFile(qualityPath, sourceCount, entries); err != nil {
		return err
	}
	manifestAbs, err := filepath.Abs(manifestPath)
	if err != nil {
		return err
	}
	printBatchSummary(batchSummary{
		SourceImages:      sourceCount,
		PhotosExtracted:   len(entries),
		QaImagesWritten:   qaImagesWritten,
		FilesMoved:        filesMoved,
		FallbackByMode:    fallbackByMode,
		WarningCount:      warnCount,
		ManifestPath:      manifestAbs,
		DebugEnabled:      *debug,
		DebugDir:          debugDir,
	})
	return nil
}

// gatherImageInputs returns absolute input paths to process, the resolved input file or
// directory (for display), and any stat/walk error. It does not create the output directory.
func gatherImageInputs(input, inputDir string, warnLog func(string)) (inputs []string, inputAbs string, err error) {
	if input != "" {
		abs, err := filepath.Abs(input)
		if err != nil {
			return nil, "", err
		}
		inputAbs = abs
		st, err := os.Stat(inputAbs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, inputAbs, fmt.Errorf("input does not exist: %s", inputAbs)
			}
			return nil, inputAbs, err
		}
		if st.IsDir() {
			return nil, inputAbs, fmt.Errorf("-input must be a file, not a directory: %s", inputAbs)
		}
		if !fsutil.IsImageName(inputAbs) {
			if warnLog != nil {
				warnLog("warning: unsupported file (expected .jpg, .jpeg, or .png), skipping: " + inputAbs)
			}
			return nil, inputAbs, nil
		}
		return []string{inputAbs}, inputAbs, nil
	}
	dir := inputDir
	if dir == "" {
		dir = "./input"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}
	inputAbs = abs
	st, err := os.Stat(inputAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, inputAbs, fmt.Errorf("input directory does not exist: %s", inputAbs)
		}
		return nil, inputAbs, err
	}
	if !st.IsDir() {
		return nil, inputAbs, fmt.Errorf("input path is not a directory: %s", inputAbs)
	}
	files, err := fsutil.ListImageFilesWithWarnings(inputAbs, warnLog)
	if err != nil {
		return nil, inputAbs, err
	}
	return files, inputAbs, nil
}

type batchSummary struct {
	SourceImages    int
	PhotosExtracted int
	QaImagesWritten int
	FilesMoved      int
	FallbackByMode  map[string]int
	WarningCount    int
	ManifestPath    string
	DebugEnabled    bool
	DebugDir        string
}

func printBatchSummary(s batchSummary) {
	fmt.Fprintln(os.Stdout, "--- batch summary ---")
	fmt.Fprintf(os.Stdout, "source images processed: %d\n", s.SourceImages)
	fmt.Fprintf(os.Stdout, "photos extracted: %d\n", s.PhotosExtracted)
	fmt.Fprintf(os.Stdout, "QA images written: %d\n", s.QaImagesWritten)
	fmt.Fprintf(os.Stdout, "files moved: %d\n", s.FilesMoved)
	fmt.Fprintln(os.Stdout, "fallback crops by mode:")
	fallbackOrder := []string{
		"axis_aligned",
		"axis_aligned_invalid_quad",
		"axis_aligned_homography_fail",
	}
	for _, m := range fallbackOrder {
		fmt.Fprintf(os.Stdout, "  %s: %d\n", m, s.FallbackByMode[m])
	}
	fmt.Fprintf(os.Stdout, "warnings: %d\n", s.WarningCount)
	fmt.Fprintf(os.Stdout, "manifest: %s\n", s.ManifestPath)
	if s.DebugEnabled {
		dbg, err := filepath.Abs(s.DebugDir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "debug directory: %s\n", s.DebugDir)
			return
		}
		fmt.Fprintf(os.Stdout, "debug directory: %s\n", dbg)
	}
}

func metaCornersToSlice(m cropper.Meta) [][]float64 {
	out := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		out[i] = []float64{m.Corners[i][0], m.Corners[i][1]}
	}
	return out
}

// cropLooksFullPageAgainstSource reports when a crop is almost the entire scan (by pixel size),
// so it must not be written as a numbered output file.
func cropLooksFullPageAgainstSource(crop image.Image, source *image.RGBA) bool {
	if source == nil || crop == nil {
		return false
	}
	sw, sh := source.Bounds().Dx(), source.Bounds().Dy()
	cw, ch := crop.Bounds().Dx(), crop.Bounds().Dy()
	if sw < 1 || sh < 1 {
		return false
	}
	return float64(cw) >= 0.9*float64(sw) && float64(ch) >= 0.9*float64(sh)
}

// outputPhotoName returns the basename for crop index n: <stem>_NNN.jpg (NNN 001-based) so a scan group sorts as
// <stem>_000_qa.jpg, <stem>_001.jpg, <stem>_002.jpg, ... The original scan filename is never reused.
func outputPhotoName(inputPath string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("internal: invalid photo index %d", n)
	}
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		return "", fmt.Errorf("empty filename stem in %q", inputPath)
	}
	name := fmt.Sprintf("%s_%03d.jpg", stem, n)
	if !isDerivedOutputPhotoName(name) {
		return "", fmt.Errorf("internal: invalid output name %q", name)
	}
	return name, nil
}

// isDerivedOutputPhotoName reports <stem>_NNN.jpg with NNN in 001..999 (lexicographic sort after <stem>_000_qa.jpg).
func isDerivedOutputPhotoName(base string) bool {
	if base == "" || !strings.EqualFold(filepath.Ext(base), ".jpg") {
		return false
	}
	s := strings.TrimSuffix(base, filepath.Ext(base))
	// not the QA file <stem>_000_qa.jpg
	if strings.HasSuffix(strings.ToLower(s), "_000_qa") {
		return false
	}
	if len(s) < 5 { // min "x_001"
		return false
	}
	if s[len(s)-4] != '_' {
		return false
	}
	suf := s[len(s)-3:]
	for j := 0; j < 3; j++ {
		if suf[j] < '0' || suf[j] > '9' {
			return false
		}
	}
	if suf == "000" {
		return false
	}
	return true
}

func saveJPEG(path string, img image.Image, quality int) error {
	return encodeJPEGFile(path, img, quality)
}

func outputQAName(inputPath string) (string, error) {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		return "", fmt.Errorf("empty filename stem in %q", inputPath)
	}
	name := fmt.Sprintf("%s_000_qa.jpg", stem)
	if !isQAOutputName(name) {
		return "", fmt.Errorf("internal: invalid QA name %q", name)
	}
	if strings.EqualFold(name, base) {
		return "", fmt.Errorf("internal: QA name must not match input: %q", name)
	}
	return name, nil
}

// isQAOutputName reports <stem>_000_qa.jpg (sorts first in a stem’s outputs).
func isQAOutputName(base string) bool {
	if base == "" || !strings.EqualFold(filepath.Ext(base), ".jpg") {
		return false
	}
	s := strings.TrimSuffix(base, filepath.Ext(base))
	return len(s) > len("_000_qa") && strings.HasSuffix(strings.ToLower(s), "_000_qa")
}

func saveQAJPEG(path string, img image.Image, quality int) error {
	return encodeJPEGFile(path, img, quality)
}

// validateAllowlistedOutputJPEG enforces that only product JPEGs are written to the output tree:
// <stem>_000_qa.jpg or <stem>_###.jpg with ### in 001..999. This blocks accidental writes of the
// original scan basename (e.g. scan.jpg) or any other disallowed name.
func validateAllowlistedOutputJPEG(base string) error {
	if base == "" {
		return fmt.Errorf("output image: empty filename")
	}
	if !strings.EqualFold(filepath.Ext(base), ".jpg") {
		return fmt.Errorf("output image %q: extension must be .jpg", base)
	}
	if isQAOutputName(base) || isDerivedOutputPhotoName(base) {
		return nil
	}
	return fmt.Errorf("disallowed output image %q: allowed patterns are <stem>_000_qa.jpg or <stem>_001.jpg through <stem>_999.jpg only", base)
}

func encodeJPEGFile(path string, img image.Image, quality int) error {
	base := filepath.Base(path)
	if err := validateAllowlistedOutputJPEG(base); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	rgba := toRGBA(img)
	if err := jpeg.Encode(f, rgba, &jpeg.Options{Quality: quality}); err != nil {
		return err
	}
	return nil
}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, b.Min, draw.Src)
	return rgba
}

func writeManifest(path string, entries []manifest.Entry) error {
	m := manifest.File{Version: 1, Entries: entries}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
