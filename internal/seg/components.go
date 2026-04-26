// Package seg: binary mask, connected components, and sort regions.
package seg

import "sort"

// Label4Connect labels foreground pixels in mask (non-zero) with 1..K using 4-connectivity; background is 0.
func Label4Connect(w, h int, mask []byte) (labels []int, numLabels int, area []int) {
	labels = make([]int, w*h)
	area = []int{0}
	next := 1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			si := y*w + x
			if mask[si] == 0 || labels[si] != 0 {
				continue
			}
			count := 0
			var q []int
			labels[si] = next
			q = append(q, si)
			for k := 0; k < len(q); k++ {
				i := q[k]
				count++
				iy, ix := i/w, i%w
				for _, d := range [4]struct{ dy, dx int }{
					{-1, 0}, {1, 0}, {0, -1}, {0, 1},
				} {
					ny, nx := iy+d.dy, ix+d.dx
					if ny < 0 || ny >= h || nx < 0 || nx >= w {
						continue
					}
					ni := ny*w + nx
					if mask[ni] == 0 || labels[ni] != 0 {
						continue
					}
					labels[ni] = next
					q = append(q, ni)
				}
			}
			for len(area) <= next {
				area = append(area, 0)
			}
			area[next] = count
			next++
		}
	}
	numLabels = next - 1
	return labels, numLabels, area
}

// Region is one connected foreground blob.
type Region struct {
	ID   int
	Area int
	Cy   float64
	Cx   float64
	MinX int
	MinY int
	MaxX int
	MaxY int
}

// BuildRegions from labels and min area; centroids are bbox center for top-left sort.
func BuildRegions(w, h int, labels []int, minArea int) []Region {
	type box struct {
		n                      int
		minX, minY, maxX, maxY int
	}
	m := make(map[int]*box)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			id := labels[y*w+x]
			if id == 0 {
				continue
			}
			b, ok := m[id]
			if !ok {
				b = &box{minX: x, minY: y, maxX: x, maxY: y, n: 0}
				m[id] = b
			}
			b.n++
			if x < b.minX {
				b.minX = x
			}
			if y < b.minY {
				b.minY = y
			}
			if x > b.maxX {
				b.maxX = x
			}
			if y > b.maxY {
				b.maxY = y
			}
		}
	}
	var out []Region
	for id, b := range m {
		if b.n < minArea {
			continue
		}
		cx := float64(b.minX+b.maxX) / 2
		cy := float64(b.minY+b.maxY) / 2
		out = append(out, Region{
			ID: id, Area: b.n, Cx: cx, Cy: cy,
			MinX: b.minX, MinY: b.minY, MaxX: b.maxX, MaxY: b.maxY,
		})
		_ = id
	}
	return out
}

// SortTopToBottomLeftToRight sort regions by reading order.
func SortTopToBottomLeftToRight(regions []Region) {
	sort.Slice(regions, func(i, j int) bool {
		ri, rj := regions[i], regions[j]
		if ri.Cy != rj.Cy {
			return ri.Cy < rj.Cy
		}
		return ri.Cx < rj.Cx
	})
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
