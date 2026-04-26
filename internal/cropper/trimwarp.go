package cropper

import (
	"image"
)

// postWarpTrimPadPx is extra border left around the tight foreground bbox after warp (2–5 px range).
const postWarpTrimPadPx = 3

// postWarpTrimMargins tightens a warped crop by thresholding (same rule as scan segmentation:
// luminance < threshold ⇒ foreground), taking the axis-aligned bbox of foreground pixels,
// expanding by postWarpTrimPadPx, and cropping. If no foreground is found, returns img unchanged.
// Bounds are always intersected with the image rectangle.
func postWarpTrimMargins(img *image.RGBA, threshold int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return img
	}
	pad := postWarpTrimPadPx
	mask := buildBinary(img, threshold)
	minX, maxX := w, -1
	minY, maxY := h, -1
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			if mask[row+x] == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return img
	}
	rect := image.Rect(
		b.Min.X+minX-pad,
		b.Min.Y+minY-pad,
		b.Min.X+maxX+1+pad,
		b.Min.Y+maxY+1+pad,
	).Intersect(b)
	if rect.Empty() || rect.Dx() < 1 || rect.Dy() < 1 {
		return img
	}
	return toRGBA(img.SubImage(rect))
}
