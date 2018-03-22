package arw

import (
	"fmt"
	"github.com/gonum/matrix/mat64"
	"log"
)

//Helper function for createToneCurve which generates a Van der Monde matrix.
func vanDerMonde(a []float64, degree int) *mat64.Dense {
	x := mat64.NewDense(len(a), degree+1, nil)
	for i := range a {
		for j, p := 0, 1.; j <= degree; j, p = j+1, p*a[i] {
			x.Set(i, j, p)
		}
	}
	return x
}

//xFactors are coefficients which are used to map incoming data to the Sony provided tone curve.
var xFactors [6]float64
var sRGBFactors [3]float64

func sRGB(x float64) float64 {
	x2 := sRGBFactors[2] * x * x
	x1 := sRGBFactors[1] * x
	x0 := sRGBFactors[0]
	val := x2 + x1 + x0
	return val
}

func gamma(x float64) float64 {
	if x > 1 {
		panic("This shouldn't be happening!" + fmt.Sprint("X=", x))
	}
	x5 := xFactors[5] * x * x * x * x * x
	x4 := xFactors[4] * x * x * x * x
	x3 := xFactors[3] * x * x * x
	x2 := xFactors[2] * x * x
	x1 := xFactors[1] * x
	x0 := xFactors[0] * 1
	val := x5 + x4 + x3 + x2 + x1 + x0 //The negative signs are already in the numbers
	return val
}

//The gamma curve points are in a 14 bit space space where we draw a curve that goes through the points.
func createToneCurve(curve [6]float64) {
	x := []float64{0, 0.2, 0.4, 0.6, 0.8, 1}
	y := []float64{float64(curve[0]), float64(curve[1]), float64(curve[2]), float64(curve[3]), curve[4], curve[5]}
	const degree = 5

	a := vanDerMonde(x, degree)
	b := mat64.NewDense(len(y), 1, y)
	c := mat64.NewDense(degree+1, 1, nil)

	qr := new(mat64.QR)
	qr.Factorize(a)

	if err := c.SolveQR(qr, false, b); err != nil {
		log.Println(err)
	}

	xFactors[5] = c.At(5, 0)
	xFactors[4] = c.At(4, 0)
	xFactors[3] = c.At(3, 0)
	xFactors[2] = c.At(2, 0)
	xFactors[1] = c.At(1, 0)
	xFactors[0] = c.At(0, 0)
}

func createSRGBCurve() {
	x := []float64{0, 0.50, 1}
	y := []float64{0, 0x2fff, 0x3fff}
	const degree = 2

	a := vanDerMonde(x, degree)
	b := mat64.NewDense(len(y), 1, y)
	c := mat64.NewDense(degree+1, 1, nil)

	qr := new(mat64.QR)
	qr.Factorize(a)

	if err := c.SolveQR(qr, false, b); err != nil {
		log.Println(err)
	}

	sRGBFactors[2] = c.At(2, 0)
	sRGBFactors[1] = c.At(1, 0)
	sRGBFactors[0] = c.At(0, 0)
}
