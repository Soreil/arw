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

func readraw14(buf []byte, rw rawDetails) *image.RGBA64 {

	sliceheader := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceheader.Len /= 2
	sliceheader.Cap /= 2
	data := *(*[]uint16)(unsafe.Pointer(&sliceheader))

	img := image.NewRGBA64(image.Rect(0, 0, int(rw.width), int(rw.height)))
	img2 := image.NewRGBA64(image.Rect(0, 0, int(rw.width), int(rw.height)))

	const factor16 = 4          //This will take us from 14 bit to 16 of value range
	const blacklevel = 512      //Taken from metadata
	const blueBalance = 1.53125 //Taken from metadata
	const greenBalance = 1.0    //Taken from metadata
	const redBalance = 2.539063 //Taken from metadata

	for i, pix := range data {
		var r, g, b uint16

		pix -= blacklevel

		if (i/int(rw.width))%2 == 0 {
			if i%2 == 0 {
				r = pix
			} else {
				g = pix
			}
		} else {
			if i%2 == 0 {
				g = pix
			} else {
				b = pix
			}
		}
		img.Set(i%int(rw.width), i/int(rw.width), color.RGBA64{r, g, b, color.Opaque.A})
	}

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			var pixel color.RGBA64

			l1 := img.RGBA64At(x, y)
			l2 := img.RGBA64At(x+1, y)
			l3 := img.RGBA64At(x, y+1)
			l4 := img.RGBA64At(x+1, y+1)

			pixel.R = uint16(float32((l1.R+l2.R+l3.R+l4.R)*factor16) * redBalance)
			pixel.G = uint16(float32(((l1.G+l2.G+l3.G+l4.G)/2)*factor16) * greenBalance)
			pixel.B = uint16(float32((l1.B+l2.B+l3.B+l4.B)*factor16) * blueBalance)
			pixel.A = color.Opaque.A

			img2.SetRGBA64(x, y, pixel)
		}
	}
	return img2
}
