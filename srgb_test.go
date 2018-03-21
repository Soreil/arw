package arw

import "testing"

func TestSRGB(t *testing.T) {
	createSRGBCurve()
	for i := 0.00; i <= 1.00; i += 0.01 {
		t.Logf("%.2f:\t%.2f\t%x", i, sRGB(i), int(sRGB(i)))
	}
}
