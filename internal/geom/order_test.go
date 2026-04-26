package geom

import (
	"math"
	"testing"
)

func assertNear(t *testing.T, a, b Point, label string) {
	t.Helper()
	if math.Abs(a.X-b.X) > 0.25 || math.Abs(a.Y-b.Y) > 0.25 {
		t.Fatalf("%s: got (%g,%g) want (%g,%g)", label, a.X, a.Y, b.X, b.Y)
	}
}

func TestOrderCornersNormalRectangle(t *testing.T) {
	tl := Point{X: 10, Y: 10}
	tr := Point{X: 100, Y: 10}
	br := Point{X: 100, Y: 200}
	bl := Point{X: 10, Y: 200}
	shuf := [4]Point{br, tl, tr, bl}
	out := OrderCornersTopLeftCCW(shuf)
	assertNear(t, out[0], tl, "TL")
	assertNear(t, out[1], tr, "TR")
	assertNear(t, out[2], br, "BR")
	assertNear(t, out[3], bl, "BL")
	if signedQuadArea2(out) <= 0 {
		t.Fatalf("expected positive signed area for dst-winding, got %v", signedQuadArea2(out))
	}
}

func TestOrderCornersRotatedRectangle(t *testing.T) {
	// 40×20 rectangle centered at (50,50), rotated ~30° (corners from unit rect scaled/rotated)
	c := Point{X: 50, Y: 50}
	w, h := 20.0, 10.0
	cos30, sin30 := math.Cos(math.Pi/6), math.Sin(math.Pi/6)
	// corners of axis rect (-w,-h)..(w,h) in local, rotate, translate
	local := [4]Point{
		{X: -w, Y: -h}, {X: w, Y: -h}, {X: w, Y: h}, {X: -w, Y: h},
	}
	var pts [4]Point
	for i, p := range local {
		x := cos30*p.X - sin30*p.Y + c.X
		y := sin30*p.X + cos30*p.Y + c.Y
		pts[i] = Point{X: x, Y: y}
	}
	// shuffle
	shuf := [4]Point{pts[2], pts[0], pts[3], pts[1]}
	out := OrderCornersTopLeftCCW(shuf)
	// TL should be topmost (min Y), then leftmost among ties
	if out[0].Y > out[1].Y || out[0].Y > out[3].Y {
		t.Fatalf("TL not topmost: %+v", out)
	}
	if ValidateQuadOrdered(out, DefaultQuadValidation()) != nil {
		t.Fatalf("ordered rotated rect should validate: %v", ValidateQuadOrdered(out, DefaultQuadValidation()))
	}
}

func TestOrderCornersSkewedQuadrilateral(t *testing.T) {
	// Convex skew quad (trapezoid-like), unordered
	tl := Point{X: 0, Y: 0}
	tr := Point{X: 100, Y: 10}
	br := Point{X: 90, Y: 80}
	bl := Point{X: 5, Y: 75}
	shuf := [4]Point{br, bl, tr, tl}
	out := OrderCornersTopLeftCCW(shuf)
	assertNear(t, out[0], tl, "TL")
	assertNear(t, out[1], tr, "TR")
	assertNear(t, out[2], br, "BR")
	assertNear(t, out[3], bl, "BL")
	if err := ValidateQuadOrdered(out, DefaultQuadValidation()); err != nil {
		t.Fatal(err)
	}
}

func TestOrderCornersUnorderedPermutation(t *testing.T) {
	base := [4]Point{
		{X: 20, Y: 30},
		{X: 200, Y: 25},
		{X: 195, Y: 180},
		{X: 15, Y: 175},
	}
	perms := [][4]int{
		{0, 1, 2, 3}, {3, 1, 0, 2}, {2, 0, 3, 1},
	}
	want := OrderCornersTopLeftCCW(base)
	for _, p := range perms {
		var sh [4]Point
		for i := 0; i < 4; i++ {
			sh[i] = base[p[i]]
		}
		got := OrderCornersTopLeftCCW(sh)
		for i := 0; i < 4; i++ {
			assertNear(t, got[i], want[i], "perm")
		}
	}
}

func TestOrderCornersConsistentWinding(t *testing.T) {
	pts := [4]Point{{10, 10}, {100, 10}, {100, 200}, {10, 200}}
	out := OrderCornersTopLeftCCW(pts)
	wantSign := signedQuadArea2([4]Point{{0, 0}, {1, 0}, {1, 1}, {0, 1}})
	gotSign := signedQuadArea2(out)
	if wantSign*gotSign <= 0 {
		t.Fatalf("winding mismatch wantSign=%v gotSign=%v", wantSign, gotSign)
	}
}
