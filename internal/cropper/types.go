package cropper

// Options for ProcessScan.
type Options struct {
	Threshold int
	MinArea   int
	Padding int
	Aspect  float64
}

// Meta per extracted photo.
type Meta struct {
	Corners    [4][2]float64
	Mode       string
	Confidence float64
}
