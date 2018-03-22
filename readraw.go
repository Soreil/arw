package arw

import (
	"image"
	"reflect"
	"unsafe"
)

func process(cur uint32, black uint32, whiteBalance float64) uint32 {
	if cur <= black {
		return 0
	} else {
		cur -= black
	}

	balanced := float64(cur) * whiteBalance
	balanced /= 0x3fff
	return uint32(sRGB(gamma(balanced)))
}

func readCRAW(buf []byte, rw rawDetails) *RGB14 {
	img := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))

	var gamma [6]float64
	gamma[0] = 0
	gamma[1] = float64(rw.gammaCurve[0])
	gamma[2] = float64(rw.gammaCurve[1])
	gamma[3] = float64(rw.gammaCurve[2])
	gamma[4] = float64(rw.gammaCurve[3])
	gamma[5] = float64(rw.gammaCurve[4])
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
					green[ir] = pixel(process(uint32(green[ir]), uint32(rw.blackLevel[2]), whiteBalanceRGGB[2]))
				}

				for ir := range blue {
					blue[ir] = pixel(process(uint32(blue[ir]), uint32(rw.blackLevel[3]), whiteBalanceRGGB[3]))
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

	var gamma [6]float64
	gamma[0] = 0
	gamma[1] = float64(rw.gammaCurve[0])
	gamma[2] = float64(rw.gammaCurve[1])
	gamma[3] = float64(rw.gammaCurve[2])
	gamma[4] = float64(rw.gammaCurve[3])
	gamma[5] = float64(rw.gammaCurve[4])
	gamma[0] /= gamma[5]
	gamma[1] /= gamma[5]
	gamma[2] /= gamma[5]
	gamma[3] /= gamma[5]
	gamma[4] /= gamma[5]
	gamma[5] /= gamma[5]

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
