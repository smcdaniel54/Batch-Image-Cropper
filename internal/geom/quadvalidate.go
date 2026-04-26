package geom

import (
	"fmt"
	"math"
)

// DefaultQuadValidation returns conservative thresholds for scan extractions (pixel units).
func DefaultQuadValidation() QuadValidation {
	return QuadValidation{
		MinArea:       400,
		MaxAspect:     25,
		MinEdgeLength: 3,
	}
}

// QuadValidation rejects degenerate or ill-conditioned quads before homography.
type QuadValidation struct {
	MinArea       float64 // minimum absolute signed area (px²)
	MaxAspect     float64 // max of width/height and height/width from edge lengths
	MinEdgeLength float64 // shortest edge must be at least this (px)
}

// ValidateQuadOrdered checks TL,TR,BR,BL order. Returns nil if the quad is acceptable.
func ValidateQuadOrdered(q [4]Point, v QuadValidation) error {
	if v.MinArea <= 0 {
		v.MinArea = 400
	}
	if v.MaxAspect <= 0 {
		v.MaxAspect = 25
	}
	if v.MinEdgeLength <= 0 {
		v.MinEdgeLength = 3
	}
	a := math.Abs(signedQuadArea2(q)) * 0.5
	if a < v.MinArea {
		return fmt.Errorf("quad area too small: %.1f < %.1f", a, v.MinArea)
	}
	edges := edgeLengths(q)
	minE := edges[0]
	maxE := edges[0]
	for _, e := range edges[1:] {
		if e < minE {
			minE = e
		}
		if e > maxE {
			maxE = e
		}
	}
	if minE < v.MinEdgeLength {
		return fmt.Errorf("edge too short: %.2f < %.2f", minE, v.MinEdgeLength)
	}
	w := math.Max(edges[0], edges[2])
	h := math.Max(edges[1], edges[3])
	if w < 1e-6 || h < 1e-6 {
		return fmt.Errorf("degenerate dimensions")
	}
	ar := w / h
	if ar > v.MaxAspect || ar < 1/v.MaxAspect {
		return fmt.Errorf("aspect ratio extreme: %.2f", ar)
	}
	if QuadSelfIntersecting(q) {
		return fmt.Errorf("self-intersecting quad")
	}
	return nil
}

func edgeLengths(q [4]Point) [4]float64 {
	var e [4]float64
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e[i] = math.Hypot(q[j].X-q[i].X, q[j].Y-q[i].Y)
	}
	return e
}

// QuadSelfIntersecting reports whether non-adjacent edges cross (bowtie).
func QuadSelfIntersecting(q [4]Point) bool {
	return segmentsIntersect(q[0], q[1], q[2], q[3]) ||
		segmentsIntersect(q[1], q[2], q[3], q[0])
}

const segEps = 1e-9

func ccw2(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func onSeg2(p, q, r Point) bool {
	return q.X <= math.Max(p.X, r.X)+segEps && q.X+segEps >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y)+segEps && q.Y+segEps >= math.Min(p.Y, r.Y)
}

// segmentsIntersect is true if closed segments ab and cd intersect at more than an endpoint-only touch
// when endpoints are distinct (non-adjacent quad edges).
func segmentsIntersect(a, b, c, d Point) bool {
	d1 := ccw2(a, b, c)
	d2 := ccw2(a, b, d)
	d3 := ccw2(c, d, a)
	d4 := ccw2(c, d, b)

	strict := func(x, y float64) bool {
		return (x > segEps && y < -segEps) || (x < -segEps && y > segEps)
	}
	if strict(d1, d2) && strict(d3, d4) {
		return true
	}
	// Collinear overlap on interior
	if math.Abs(d1) < segEps && onSeg2(a, c, b) {
		return !(nearPoint(c, a) || nearPoint(c, b))
	}
	if math.Abs(d2) < segEps && onSeg2(a, d, b) {
		return !(nearPoint(d, a) || nearPoint(d, b))
	}
	if math.Abs(d3) < segEps && onSeg2(c, a, d) {
		return !(nearPoint(a, c) || nearPoint(a, d))
	}
	if math.Abs(d4) < segEps && onSeg2(c, b, d) {
		return !(nearPoint(b, c) || nearPoint(b, d))
	}
	return false
}

func nearPoint(p, q Point) bool {
	return math.Abs(p.X-q.X) < segEps && math.Abs(p.Y-q.Y) < segEps
}
