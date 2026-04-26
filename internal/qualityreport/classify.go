package qualityreport

// ClassifyStatus returns the batch quality line given per–output-file confidence scores.
// Rules (first match in priority order is not how we do it; instead):
//   - FAIL: if any confidence <= 0.2
//   - PASS: if all confidences are >= 0.65
//   - REVIEW: otherwise (e.g. any value in (0.2, 0.65))
//
// If confidences is empty, returns PASS (nothing failed the high bar).
func ClassifyStatus(confidences []float64) string {
	for _, c := range confidences {
		if c <= 0.2 {
			return "FAIL"
		}
	}
	if len(confidences) == 0 {
		return "PASS"
	}
	for _, c := range confidences {
		if c < 0.65 {
			return "REVIEW"
		}
	}
	return "PASS"
}
