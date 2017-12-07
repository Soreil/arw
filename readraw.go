package arw

import (
	"github.com/gonum/matrix/mat64"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"reflect"
	"unsafe"
)

type rawDetails struct {
	width         uint16
	height        uint16
	bitDepth      uint16
	rawType       sonyRawFile
	offset        uint32
	stride        uint32
	length        uint32
	blackLevel    [4]uint16
	WhiteBalance  [4]int16
	gammaCurve    [5]uint16
	crop          image.Rectangle
	cfaPattern    [4]uint8 //TODO(sjon): This might not always be 4 bytes is my suspicion. We currently take from the offset
	cfaPatternDim [2]uint16
}

func extractDetails(rs io.ReadSeeker) (rawDetails, error) {
	var rw rawDetails

	header, err := ParseHeader(rs)
	meta, err := ExtractMetaData(rs, int64(header.Offset), 0)
	if err != nil {
		return rw, err
	}

	for _, fia := range meta.FIA {
		if fia.Tag != SubIFDs {
			continue
		}

		rawIFD, err := ExtractMetaData(rs, int64(fia.Offset), 0)
		if err != nil {
			return rw, err
		}

		for i, v := range rawIFD.FIA {
			switch v.Tag {
			case ImageWidth:
				rw.width = uint16(v.Offset)
			case ImageHeight:
				rw.height = uint16(v.Offset)
			case BitsPerSample:
				rw.bitDepth = uint16(v.Offset)
			case SonyRawFileType:
				rw.rawType = sonyRawFile(v.Offset)
			case StripOffsets:
				rw.offset = v.Offset
			case RowsPerStrip:
				rw.stride = v.Offset / 2
			case StripByteCounts:
				rw.length = v.Offset
			case SonyCurve:
				curve := *rawIFD.FIAvals[i].short
				copy(rw.gammaCurve[:4], curve)
				rw.gammaCurve[4] = 0x3fff
			case BlackLevel2:
				black := *rawIFD.FIAvals[i].short
				copy(rw.blackLevel[:], black)
			case WB_RGGBLevels:
				balance := *rawIFD.FIAvals[i].sshort
				copy(rw.WhiteBalance[:], balance)
			case DefaultCropSize:
			case CFAPattern2:
				rw.cfaPattern[0] = uint8((v.Offset & 0x000000ff) >> 0)
				rw.cfaPattern[1] = uint8((v.Offset & 0x0000ff00) >> 8)
				rw.cfaPattern[2] = uint8((v.Offset & 0x00ff0000) >> 16)
				rw.cfaPattern[3] = uint8((v.Offset & 0xff000000) >> 24)
			case CFARepeatPatternDim:
				rw.cfaPatternDim[0] = uint16((v.Offset * 0x0000ffff) >> 0)
				rw.cfaPatternDim[1] = uint16((v.Offset * 0xffff0000) >> 16)
			}
		}
	}
	return rw, nil
}

//Helper function for gammacorrect
func vandermonde(a []float64, degree int) *mat64.Dense {
	x := mat64.NewDense(len(a), degree+1, nil)
	for i := range a {
		for j, p := 0, 1.; j <= degree; j, p = j+1, p*a[i] {
			x.Set(i, j, p)
		}
	}
	return x
}

//This function is created by gammacorrect
var gamma func(uint32) uint32

//The gamma curve points are in a 14 bit space space where we draw a curve that goes through the points.
func gammacorrect(curve [4]uint32) {
	x := []float64{0, 1, 2, 3, 4, 5} // It would be nice if we could make this [0,0x3fff] but that seems to be impossible

	y := []float64{0, float64(curve[0]), float64(curve[1]), float64(curve[2]), float64(curve[3]), 0x3fff}
	const degree = 5

	a := vandermonde(x, degree)
	b := mat64.NewDense(len(y), 1, y)
	c := mat64.NewDense(degree+1, 1, nil)

	qr := new(mat64.QR)
	qr.Factorize(a)

	if err := c.SolveQR(qr, false, b); err != nil {
		log.Println(err)
	}

	gamma = func(g uint32) uint32 {
		if g > 0x3fff {
			return 0x3fff //TODO(sjon): Should it be concidered a bug if we receive blown out values here?
		}
		x := (float64(g) / 0x3fff) * 5 //We need to keep x in between 0 and 5, this maps to 0x0 to 0x3fff
		x5 := c.At(5, 0) * math.Pow(x, 5)
		x4 := c.At(4, 0) * math.Pow(x, 4)
		x3 := c.At(3, 0) * math.Pow(x, 3)
		x2 := c.At(2, 0) * math.Pow(x, 2)
		x1 := c.At(1, 0) * math.Pow(x, 1)
		x0 := c.At(0, 0) * math.Pow(x, 0)
		val := x5 + x4 + x3 + x2 + x1 + x0 //The negative signs are already in the numbers
		if val > 0x3fff {
			//panic("unexpectedly high gamma conversion result")
		}
		return uint32(val)
	}
}

func process(cur uint32, black uint32, whitebalance uint32) uint32 {
	if cur <= black {
		return cur
	} else {
		cur -= black
	}

	cur = (cur * whitebalance) / 1024
	cur = gamma(cur)
	return cur
}

func readraw14(buf []byte, rw rawDetails) *RGB14 {
	sliceheader := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceheader.Len /= 2
	sliceheader.Cap /= 2
	data := *(*[]uint16)(unsafe.Pointer(&sliceheader))

	img := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))

	var cur uint32
	var gamma [4]uint32
	gamma[0] = uint32(rw.gammaCurve[0])
	gamma[1] = uint32(rw.gammaCurve[1])
	gamma[2] = uint32(rw.gammaCurve[2])
	gamma[3] = uint32(rw.gammaCurve[3])
	gammacorrect(gamma)

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[0]), uint32(rw.WhiteBalance[0]))
			img.Pix[y*img.Stride+x].R = uint16(cur)
			x++

			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[1]), uint32(rw.WhiteBalance[1]))
			img.Pix[y*img.Stride+x].G = uint16(cur)
		}
		y++

		for x := 0; x < img.Rect.Max.X; x++ {
			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[2]), uint32(rw.WhiteBalance[2]))
			img.Pix[y*img.Stride+x].G = uint16(cur)
			x++

			cur = uint32(data[y*img.Stride+x])
			cur = process(cur, uint32(rw.blackLevel[3]), uint32(rw.WhiteBalance[3]))
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

// NewRGBA returns a new RGBA image with the given bounds.
func NewRGB14(r image.Rectangle) *RGB14 {
	w, h := r.Dx(), r.Dy()
	buf := make([]pixel16, w*h)
	return &RGB14{buf, w, r}
}

// RGBA64 is an in-memory image whose At method returns pixel16 values.
type RGB14 struct {
	Pix []pixel16
	// Stride is the Pix stride between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (r *RGB14) at(x, y int) pixel16 {
	return r.Pix[(y*r.Stride)+x]
}

func (r *RGB14) At(x, y int) color.Color {
	return r.at(x, y)
}

func (r *RGB14) Bounds() image.Rectangle {
	return r.Rect.Bounds()
}

func (r *RGB14) ColorModel() color.Model {
	return color.RGBA64Model
}

func (c pixel16) RGBA() (r, g, b, a uint32) {

	return uint32(c.R) * 4, uint32(c.G) * 4, uint32(c.B) * 4, 0xffff

}

func (r *RGB14) set(x, y int, pixel pixel16) {
	r.Pix[y*r.Stride+x] = pixel
}

type pixel16 struct {
	R uint16
	G uint16
	B uint16
	_ uint16
}
