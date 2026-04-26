package contour

import (
	"math"

	"batch-image-cropper/internal/geom"
)

// MinAreaRectBrute returns the minimum-area enclosing rectangle of an arbitrary point set
// by scanning rotation angles. Corners in CCW in image space (may need geom.Order for TL..).
func MinAreaRectBrute(pts []geom.Point) (corners [4]geom.Point, area float64) {
	if len(pts) == 0 {
		return corners, 1e18
	}
	var cx, cy float64
	for _, p := range pts {
		cx += p.X
		cy += p.Y
	}
	cx /= float64(len(pts))
	cy /= float64(len(pts))
	bestA := 1e18
	var bestC, bestS float64
	for t := 0.0; t < math.Pi*0.9999; t += math.Pi / 720.0 { // 0.25°
		c, s := math.Cos(t), math.Sin(t)
		var minX, maxX, minY, maxY float64
		for i, p := range pts {
			xo := p.X - cx
			yo := p.Y - cy
			xr := c*xo + s*yo
			yr := -s*xo + c*yo
			if i == 0 {
				minX, maxX, minY, maxY = xr, xr, yr, yr
			} else {
				if xr < minX {
					minX = xr
				}
				if xr > maxX {
					maxX = xr
				}
				if yr < minY {
					minY = yr
				}
				if yr > maxY {
					maxY = yr
				}
			}
		}
		a := (maxX - minX) * (maxY - minY)
		if a < bestA {
			bestA = a
			bestC, bestS = c, s
		}
	}
	// back corners in rotated min frame, then to image
	// min corner in local (minX, minY) after rotation: inverse rotate
	c, s := bestC, bestS
	// recompute min in best frame
	var minX, maxX, minY, maxY float64
	for i, p := range pts {
		xo := p.X - cx
		yo := p.Y - cy
		xr := c*xo + s*yo
		yr := -s*xo + c*yo
		if i == 0 {
			minX, maxX, minY, maxY = xr, xr, yr, yr
		} else {
			if xr < minX {
				minX = xr
			}
			if xr > maxX {
				maxX = xr
			}
			if yr < minY {
				minY = yr
			}
			if yr > maxY {
				maxY = yr
			}
		}
	}
	// 4 local corners in rotated frame: x' right, y' down
	local := [4]struct{ x, y float64 }{
		{minX, minY},
		{maxX, minY},
		{maxX, maxY},
		{minX, maxY},
	}
	// inverse R^T: [dx,dy] = R^T [lx,ly] with R: x' = c*xo + s*yo, y' = -s*xo + c*yo
	//            => xo = c*lx - s*ly, yo = s*lx + c*ly
	for k := 0; k < 4; k++ {
		lx, ly := local[k].x, local[k].y
		dx := c*lx - s*ly
		dy := s*lx + c*ly
		corners[k] = geom.Point{X: dx + cx, Y: dy + cy}
	}
	if bestA <= 0 {
		bestA = 1e-6
	}
	return corners, bestA
}

// AxisAlignedBox returns (bl,tr) style corners as tl,tr,br,bl for quad
func AxisAlignedBox(minX, minY, maxX, maxY float64) [4]geom.Point {
	return [4]geom.Point{
		{X: minX, Y: minY},
		{X: maxX, Y: minY},
		{X: maxX, Y: maxY},
		{X: minX, Y: maxY},
	}
}
