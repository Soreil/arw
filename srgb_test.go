package arw

import (
	"fmt"
	"testing"
)

func TestSRGB(t *testing.T) {
	createSRGBCurve()
	var results []float64
	for i := 0.00; i <= 1.00; i += 0.01 {
		//	t.Logf("%.2f:\t%.2f\t%x", i, sRGB(i), int(sRGB(i)))
		results = append(results, sRGB(i))
	}
	fmt.Println("# x y")
	for i, v := range results {
		fmt.Printf("%.2v %#x\n", float64(i)*0.01, int(v))
	}
}

func TestToneCurve(t *testing.T) {
	createToneCurve([6]float64{0, 8000, 10400, 12900, 14100, 0x3fff})
	var results []float64
	for i := 0.00; i <= 1.00; i += 0.01 {
		//t.Logf("%d:\t%.2f\t%x", i, gamma(float64(i)), int(gamma(float64(i))))
		results = append(results, gamma(float64(i)))
	}

	fmt.Println("# x y")
	for i, v := range results {
		fmt.Println(0.01*float64(i), v)
	}
}
