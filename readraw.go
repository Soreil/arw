package arw

import (
	"github.com/gonum/matrix/mat64"
	"image"
	"log"
	"reflect"
	"unsafe"
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
var sRGBFactors [2]float64

func sRGB(x float64) float64 {
	x1 := sRGBFactors[1] * x
	x0 := sRGBFactors[0]
	val := x1 + x0
	return val
}

func gamma(xINT float64) float64 {
	if xINT > 0x3fff {
		panic("This shouldn't be happening!")
		return 0x3fff //TODO(sjon): Should it be considered a bug if we receive blown out values here?
	}
	/*TODO(sjon): What should the correct value be here? Lower values seem to work for most inputs
	the building sample works ok with 0x200 but the baloon sample clips in multiple places with values lower than 0xcc
	*/
	x := xINT / 0x0ccc //We need to keep x in between 0 and 5, this maps to 0x0 to 0x3fff

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
func createToneCurve(curve [4]uint32) {
	x := []float64{0, 1, 2, 3, 4, 5} // It would be nice if we could make this [0,0x3fff] but that seems to be impossible

	y := []float64{0, float64(curve[0]), float64(curve[1]), float64(curve[2]), float64(curve[3]), 0x3fff}
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
	x := []float64{0, 0.25, 1}
	y := []float64{0, 0x2fff, 0x3fff}
	const degree = 1

	a := vanDerMonde(x, degree)
	b := mat64.NewDense(len(y), 1, y)
	c := mat64.NewDense(degree+1, 1, nil)

	qr := new(mat64.QR)
	qr.Factorize(a)

	if err := c.SolveQR(qr, false, b); err != nil {
		log.Println(err)
	}

	sRGBFactors[1] = c.At(1, 0)
	sRGBFactors[0] = c.At(0, 0)
}

func process(cur uint32, black uint32, whiteBalance float64) uint32 {
	if cur <= black {
		return 0
	} else {
		cur -= black
	}

	balanced := float64(cur) * whiteBalance
	return uint32(sRGB(gamma(balanced) / 0x3fff))
}

func readCRAW(buf []byte, rw rawDetails) *RGB14 {
	img := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))

	var gamma [4]uint32
	gamma[0] = uint32(rw.gammaCurve[0])
	gamma[1] = uint32(rw.gammaCurve[1])
	gamma[2] = uint32(rw.gammaCurve[2])
	gamma[3] = uint32(rw.gammaCurve[3])
	createToneCurve(gamma)

	var whiteBalanceRGGB [4]float64
	var maxBalance int16
	if rw.WhiteBalance[0] > rw.WhiteBalance[1] {
		maxBalance = rw.WhiteBalance[0]
	} else {
		maxBalance = rw.WhiteBalance[1]
	}
	if rw.WhiteBalance[2] > maxBalance {
		maxBalance = rw.WhiteBalance[2]
	}
	if rw.WhiteBalance[3] > maxBalance {
		maxBalance = rw.WhiteBalance[3]
	}

	whiteBalanceRGGB[0] = float64(rw.WhiteBalance[0]) / float64(maxBalance)
	whiteBalanceRGGB[1] = float64(rw.WhiteBalance[1]) / float64(maxBalance)
	whiteBalanceRGGB[2] = float64(rw.WhiteBalance[2]) / float64(maxBalance)
	whiteBalanceRGGB[3] = float64(rw.WhiteBalance[3]) / float64(maxBalance)

	log.Println(whiteBalanceRGGB)

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x += 32 {
			if y%2 == 0 {
				base := y*img.Stride + x

				//fmt.Printf("Red block on line: %v\t column: %v\n", y, x)
				block := readCrawBlock(buf[base : base+pixelBlockSize]) //16 red pixels, inverleaved with following 16 green
				red := block.Decompress()

				//fmt.Printf("Green block on line: %v\t column: %v\n", y, x+pixelBlockSize)
				block = readCrawBlock(buf[base+pixelBlockSize : base+pixelBlockSize+pixelBlockSize]) // idem
				green := block.Decompress()

				for ir := range red {
					red[ir] = pixel(process(uint32(red[ir]), uint32(rw.blackLevel[0]), whiteBalanceRGGB[0]))
				}

				for ir := range green {
					green[ir] = pixel(process(uint32(green[ir]), uint32(rw.blackLevel[1]), whiteBalanceRGGB[1]))
				}
				for i := 0; i < pixelBlockSize; i++ {
					img.Pix[base+(i*2)].R = uint16(red[i])
					img.Pix[base+(i*2)+1].G = uint16(green[i])
				}
			} else {
				//fmt.Printf("Green block on line: %v\t column: %v\n", y, x)
				base := y*img.Stride + x

				block := readCrawBlock(buf[base : base+pixelBlockSize]) //16 red pixels, inverleaved with following 16 green
				green := block.Decompress()

				//fmt.Printf("Green block on line: %v\t column: %v\n", y, x+pixelBlockSize)
				block = readCrawBlock(buf[base+pixelBlockSize : base+pixelBlockSize+pixelBlockSize]) // idem
				blue := block.Decompress()

				for ir := range green {
					green[ir] = pixel(process(uint32(green[ir]), uint32(rw.blackLevel[0]), whiteBalanceRGGB[0]))
				}

				for ir := range blue {
					blue[ir] = pixel(process(uint32(blue[ir]), uint32(rw.blackLevel[1]), whiteBalanceRGGB[1]))
				}
				for i := 0; i < pixelBlockSize; i++ {
					img.Pix[base+(i*2)].G = uint16(green[i])
					img.Pix[base+(i*2)+1].B = uint16(blue[i])
				}
			}
		}
	}
	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			img.Pix[y*img.Stride+x].G = img.Pix[y*img.Stride+x+1].G
			img.Pix[y*img.Stride+x].B = img.Pix[(y+1)*img.Stride+x+1].B
			x++
			img.Pix[y*img.Stride+x].R = img.Pix[y*img.Stride+x-1].R
			img.Pix[y*img.Stride+x].B = img.Pix[(y+1)*img.Stride+x].B
		}
		y++

		for x := 0; x < img.Rect.Max.X; x++ {
			img.Pix[y*img.Stride+x].R = img.Pix[(y-1)*img.Stride+x].R
			img.Pix[y*img.Stride+x].B = img.Pix[y*img.Stride+x+1].B
			x++
			img.Pix[y*img.Stride+x].R = img.Pix[(y-1)*img.Stride+x-1].R
			img.Pix[y*img.Stride+x].G = img.Pix[y*img.Stride+x-1].G
		}
	}

	return img
}

func readRaw14(buf []byte, rw rawDetails) *RGB14 {
	//Since we are working with 14 it bytes we choose to simply change the slice's header
	sliceHeader := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceHeader.Len /= 2
	sliceHeader.Cap /= 2
	data := *(*[]uint16)(unsafe.Pointer(&sliceHeader))

	img := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))

	var cur uint32

	var gamma [4]uint32
	gamma[0] = uint32(rw.gammaCurve[0])
	gamma[1] = uint32(rw.gammaCurve[1])
	gamma[2] = uint32(rw.gammaCurve[2])
	gamma[3] = uint32(rw.gammaCurve[3])
	createToneCurve(gamma)
	createSRGBCurve()

	var whiteBalanceRGGB [4]float64
	var maxBalance int16
	if rw.WhiteBalance[0] > rw.WhiteBalance[1] {
		maxBalance = rw.WhiteBalance[0]
	} else {
		maxBalance = rw.WhiteBalance[1]
	}
	if rw.WhiteBalance[2] > maxBalance {
		maxBalance = rw.WhiteBalance[2]
	}
	if rw.WhiteBalance[3] > maxBalance {
		maxBalance = rw.WhiteBalance[3]
	}

	whiteBalanceRGGB[0] = float64(rw.WhiteBalance[0]) / float64(maxBalance)
	whiteBalanceRGGB[1] = float64(rw.WhiteBalance[1]) / float64(maxBalance)
	whiteBalanceRGGB[2] = float64(rw.WhiteBalance[2]) / float64(maxBalance)
	whiteBalanceRGGB[3] = float64(rw.WhiteBalance[3]) / float64(maxBalance)

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[0]), whiteBalanceRGGB[0])
			img.Pix[y*img.Stride+x].R = uint16(cur)
			x++

			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[1]), whiteBalanceRGGB[1])
			img.Pix[y*img.Stride+x].G = uint16(cur)
		}
		y++

		for x := 0; x < img.Rect.Max.X; x++ {
			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[2]), whiteBalanceRGGB[2])
			img.Pix[y*img.Stride+x].G = uint16(cur)
			x++

			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[3]), whiteBalanceRGGB[3])
			img.Pix[y*img.Stride+x].B = uint16(cur)
		}
	}

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			img.Pix[y*img.Stride+x].G = img.Pix[y*img.Stride+x+1].G
			img.Pix[y*img.Stride+x].B = img.Pix[(y+1)*img.Stride+x+1].B
			x++
			img.Pix[y*img.Stride+x].R = img.Pix[y*img.Stride+x-1].R
			img.Pix[y*img.Stride+x].B = img.Pix[(y+1)*img.Stride+x].B
		}
		y++

		for x := 0; x < img.Rect.Max.X; x++ {
			img.Pix[y*img.Stride+x].R = img.Pix[(y-1)*img.Stride+x].R
			img.Pix[y*img.Stride+x].B = img.Pix[y*img.Stride+x+1].B
			x++
			img.Pix[y*img.Stride+x].R = img.Pix[(y-1)*img.Stride+x-1].R
			img.Pix[y*img.Stride+x].G = img.Pix[y*img.Stride+x-1].G
		}
	}

	return img
}
