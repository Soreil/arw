package arw

import (
	"image"
	"image/color"
	"io"
	"reflect"
	"unsafe"
)

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

func readraw14(buf []byte, rw rawDetails) *RGB14 {
	sliceheader := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceheader.Len /= 2
	sliceheader.Cap /= 2
	data := *(*[]uint16)(unsafe.Pointer(&sliceheader))

	img := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))
	img2 := NewRGB14(image.Rect(0, 0, int(rw.width), int(rw.height)))

	const blackLevel = 512      //Taken from metadata
	const blueBalance = 1.53125 //Taken from metadata
	const greenBalance = 1.0    //Taken from metadata
	const redBalance = 2.539063 //Taken from metadata

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			img.Pix[y*img.Stride+x].R = data[y*img.Stride+x] - blackLevel
			x++
			img.Pix[y*img.Stride+x].G = data[y*img.Stride+x] - blackLevel
		}
		y++

		for x := 0; x < img.Rect.Max.X; x++ {
			var p pixel16
			pix := data[y*img.Stride+x]
			pix -= blackLevel
			p.G = pix
			img.Pix[y*img.Stride+x].G = data[y*img.Stride+x] - blackLevel
			x++

			var p2 pixel16
			pix = data[y*img.Stride+x]
			pix -= blackLevel
			p2.B = pix
			img.Pix[y*img.Stride+x].B = data[y*img.Stride+x] - blackLevel
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
