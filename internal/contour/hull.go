package contour

import (
	"math"
	"sort"

	"photo-cropper/internal/geom"
)

// ConvexHullMonotone returns vertices of convex hull in CCW order (no collinear duplicate in minimal form).
func ConvexHullMonotone(pts []geom.Point) []geom.Point {
	if len(pts) < 3 {
		return append([]geom.Point{}, pts...)
	}
	sort.Slice(pts, func(i, j int) bool {
		if pts[i].X != pts[j].X {
			return pts[i].X < pts[j].X
		}
		return pts[i].Y < pts[j].Y
	})
	lower := make([]geom.Point, 0, len(pts))
	for _, p := range pts {
		for len(lower) >= 2 && cross(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}
	upper := make([]geom.Point, 0, len(pts))
	for i := len(pts) - 1; i >= 0; i-- {
		p := pts[i]
		for len(upper) >= 2 && cross(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}
	hull := append(lower[:len(lower)-1], upper[:len(upper)-1]...)
	return hull
}

func cross(o, a, b geom.Point) float64 {
	return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
}

// PolygonArea returns signed area (positive for CCW in standard math, image y-down may differ).
func PolygonArea(pts []geom.Point) float64 {
	var a float64
	for i := 0; i < len(pts); i++ {
		j := (i + 1) % len(pts)
		a += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	return 0.5 * a
}

// IsConvex checks simple quad
func IsConvex(pts [4]geom.Point) bool {
	var signs int
	for i := 0; i < 4; i++ {
		a, b, c := pts[i], pts[(i+1)%4], pts[(i+2)%4]
		cr := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if cr > 1e-6 {
			signs |= 1
		} else if cr < -1e-6 {
			signs |= 2
		}
	}
	return signs != 3
}

// DistPointSegment distance from p to segment ab
func DistPointSegment(p, a, b geom.Point) float64 {
	vx, vy := b.X-a.X, b.Y-a.Y
	wx, wy := p.X-a.X, p.Y-a.Y
	c1 := wx*vx + wy*vy
	if c1 <= 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	c2 := vx*vx + vy*vy
	if c2 <= c1 {
		return math.Hypot(p.X-b.X, p.Y-b.Y)
	}
	t := c1 / c2
	projX := a.X + t*vx
	projY := a.Y + t*vy
	return math.Hypot(p.X-projX, p.Y-projY)
}
