// Package geom has point ordering and image-space helpers.
package geom

import (
	"image"
	"math"
	"sort"
)

// Point is a 2D point in image coordinates (origin top-left, y down).
type Point struct {
	X, Y float64
}

// OrderCornersTopLeftCCW reorders four arbitrary corners to TL, TR, BR, BL for perspective warp
// (dst rectangle TL=(0,0), TR=(w,0), BR=(w,h), BL=(0,h)).
//
// Method: centroid → sort by polar angle atan2(y−cy, x−cx) ascending (clockwise walk in y-down space) →
// rotate so the image top-left corner (min Y, then min X) is first → fix winding to match the dst rectangle.
func OrderCornersTopLeftCCW(pts [4]Point) [4]Point {
	c := Centroid(pts)
	type idxAng struct {
		i   int
		ang float64
	}
	var arr [4]idxAng
	for i := 0; i < 4; i++ {
		arr[i] = idxAng{
			i:   i,
			ang: math.Atan2(pts[i].Y-c.Y, pts[i].X-c.X),
		}
	}
	sort.Slice(arr[:], func(a, b int) bool {
		if arr[a].ang != arr[b].ang {
			return arr[a].ang < arr[b].ang
		}
		return arr[a].i < arr[b].i
	})
	var sorted [4]Point
	for j := 0; j < 4; j++ {
		sorted[j] = pts[arr[j].i]
	}
	// Cyclic order is consistent; pick rotation so TL = topmost then leftmost (image coordinates).
	best := 0
	for i := 1; i < 4; i++ {
		if isMoreTopLeft(sorted[i], sorted[best]) {
			best = i
		}
	}
	rot := rotateCycle(sorted, best)
	// Match winding of unit dst quad (0,0),(1,0),(1,1),(0,1) → positive signed area.
	want := signedQuadArea2([4]Point{{0, 0}, {1, 0}, {1, 1}, {0, 1}})
	got := signedQuadArea2(rot)
	if want*got < 0 {
		rot = [4]Point{rot[0], rot[3], rot[2], rot[1]}
	}
	return rot
}

func isMoreTopLeft(a, b Point) bool {
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.X < b.X
}

func rotateCycle(p [4]Point, start int) [4]Point {
	var out [4]Point
	for j := 0; j < 4; j++ {
		out[j] = p[(start+j)%4]
	}
	return out
}

// signedQuadArea2 is the shoelace sum ∑(xi·yi+1 − xi+1·yi) for TL→TR→BR→BL (twice signed area).
func signedQuadArea2(q [4]Point) float64 {
	var s float64
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		s += q[i].X*q[j].Y - q[j].X*q[i].Y
	}
	return s
}

// Centroid of 4 points.
func Centroid(pts [4]Point) Point {
	var c Point
	for _, p := range pts {
		c.X += p.X
		c.Y += p.Y
	}
	c.X /= 4
	c.Y /= 4
	return c
}

// ExpandQuadFromCenter moves each corner away from c by a scale factor s = 1 + 2*pad/minEdge.
func ExpandQuadFromCenter(pts [4]Point, c Point, padding float64) [4]Point {
	if padding <= 0 {
		return pts
	}
	var minE float64 = 1e9
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := math.Hypot(pts[j].X-pts[i].X, pts[j].Y-pts[i].Y)
		if e < minE {
			minE = e
		}
	}
	if minE < 1 {
		minE = 1
	}
	s := 1.0 + 2.0*padding/minE
	var out [4]Point
	for i := 0; i < 4; i++ {
		dx := pts[i].X - c.X
		dy := pts[i].Y - c.Y
		out[i] = Point{X: c.X + dx*s, Y: c.Y + dy*s}
	}
	return out
}

// ImagePointToFloat is a small helper.
func ImagePointToFloat(p image.Point) Point {
	return Point{X: float64(p.X), Y: float64(p.Y)}
}
