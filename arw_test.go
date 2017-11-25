package arw

import (
	"image/jpeg"
	"os"
	"testing"
	"time"
	"fmt"
	"bytes"
	"io/ioutil"
	"image"
	"image/color"
	"reflect"
	"unsafe"
	"image/png"
)

const testFileLocation = "samples"

var samples map[sonyRawFile][]string

func init() {
	os.Chdir(testFileLocation)
	samples = make(map[sonyRawFile][]string)
	samples[raw14] = append(samples[raw14],`Y-a7r-iii-DSC00024`)
	samples[raw12] = append(samples[raw12],`DSC01373`)
	samples[craw] = append(samples[craw],`1`)
	samples[crawLossless] = append(samples[crawLossless],`DSC01373`)
}

type rawDetails struct {
	width       uint16
	height      uint16
	bitDepth    uint16
	rawType     sonyRawFile
	offset      uint32
	stride      uint32
	length      uint32
	blackLevel [4]uint16
	WhiteBalance [4]int16
	gammaCurve [5]uint16
	crop image.Rectangle
	cfaPattern [4]uint8 //TODO(sjon): This might not always be 4 bytes is my suspicion. We currently take from the offset
	cfaPatternDim [2]uint16
}

func TestDecodeA7R3(t *testing.T) {
	samplename := samples[raw14][0]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}
	header, err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW, int64(header.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	var rw rawDetails

	for _, fia := range meta.FIA {
		if fia.Tag != SubIFDs {
			continue
		}

		rawIFD, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
		if err != nil {
			t.Error(err)
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
				rw.stride = v.Offset/2
			case StripByteCounts:
				rw.length = v.Offset
			case SonyCurve:
				curve := *rawIFD.FIAvals[i].short
				copy(rw.gammaCurve[:4],curve)
				rw.gammaCurve[4] = 0x3fff
			case BlackLevel2:
				black := *rawIFD.FIAvals[i].short
				copy(rw.blackLevel[:],black)
			case WB_RGGBLevels:
				balance := *rawIFD.FIAvals[i].sshort
				copy(rw.WhiteBalance[:],balance)
			case DefaultCropSize:
			case CFAPattern2:
				rw.cfaPattern[0] = uint8((v.Offset&0x000000ff)>>0)
				rw.cfaPattern[1] = uint8((v.Offset&0x0000ff00)>>8)
				rw.cfaPattern[2] = uint8((v.Offset&0x00ff0000)>>16)
				rw.cfaPattern[3] = uint8((v.Offset&0xff000000)>>24)
			case CFARepeatPatternDim:
				rw.cfaPatternDim[0] = uint16((v.Offset*0x0000ffff)>>0)
				rw.cfaPatternDim[1] = uint16((v.Offset*0xffff0000)>>16)
			}
		}
	}
	t.Logf("%+v\n",rw)

	buf := make([]byte,rw.length)
	testARW.ReadAt(buf,int64(rw.offset))

	if rw.rawType != raw14 {
		t.Error("Not yet implemented type:",rw.rawType)
	}

	sliceheader := *(*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sliceheader.Len /= 2
	sliceheader.Cap /= 2
	data := *(*[]uint16)(unsafe.Pointer(&sliceheader))

	img := image.NewRGBA64(image.Rect(0,0,int(rw.width),int(rw.height)))
	img2 := image.NewRGBA64(image.Rect(0,0,int(rw.width),int(rw.height)))

	const factor16 = 4
	const blacklevel = 512
	const blueBalance = 1.53125
	const greenBalance = 1.0
	const redBalance = 2.539063

	for i,pix := range data {
		var r,g,b uint16

		pix -=blacklevel

		if (i / int(rw.width)) % 2 == 0 {
			if i % 2 == 0 {
				r = pix
			} else {
				g = pix
			}
		} else {
			if i % 2 == 0 {
				g = pix
			} else {
				b = pix
			}
		}
		img.Set(i%int(rw.width),i/int(rw.width),color.RGBA64{r,g,b,color.Opaque.A})
	}

	for y := 0; y < img.Rect.Max.Y; y++ {
		for x := 0; x < img.Rect.Max.X; x++ {
			var pixel color.RGBA64

			l1 := img.RGBA64At(x,y)
			l2 := img.RGBA64At(x+1,y)
			l3 := img.RGBA64At(x,y+1)
			l4 := img.RGBA64At(x+1,y+1)

			pixel.R = uint16(float32((l1.R +l2.R +l3.R +l4.R)*factor16)*redBalance)
			pixel.G = uint16(float32(((l1.G +l2.G +l3.G +l4.G)/2)*factor16)*greenBalance)
			pixel.B = uint16(float32((l1.B +l2.B +l3.B +l4.B)*factor16)*blueBalance)
			pixel.A = color.Opaque.A

			img2.SetRGBA64(x,y,pixel)
		}
	}

	const prefix = "A7R3Black"
	os.Chdir("experiments")

	if false {
		jpgName := prefix + fmt.Sprint(time.Now().Unix()) + ".jpg"
		f, err := os.Create(jpgName)
		if err != nil {
			t.Error(err)
		}

		jpeg.Encode(f, img2, nil)

		f.Close()
	}

	if true {
		display(img)
	}

	if false {
		f,err := os.Create(prefix+fmt.Sprint(time.Now().Unix())+".png")
		if err != nil {
			t.Error(err)
		}

		png.Encode(f,img2)

		f.Close()
	}
}

func TestMetadata(t *testing.T) {
	samplename := samples[raw14][0]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}

	header, err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW, int64(header.Offset), 0)
	if err != nil {
		t.Error(err)
	}
	t.Log("0th IFD for primary image data")
	t.Log(meta)

	for _, v := range meta.FIA {
		t.Logf("%+v\n", v)
	}

	for _, fia := range meta.FIA {
		if fia.Tag == SubIFDs {
			t.Log("Reading subIFD located at: ", fia.Offset)
			next, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
			if err != nil {
				t.Error(err)
			}
			t.Log("A subIFD, who knows what we'll find here!")
			t.Log(next)
			for _, v := range next.FIA {
				t.Logf("%+v\n", v)
			}

		}

		if fia.Tag == GPSTag {
			gps, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
			if err != nil {
				t.Error(err)
			}

			t.Log("GPS IFD (GPS Info Tag)")
			t.Log(gps)
		}

		if fia.Tag == ExifTag {
			exif, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
			if err != nil {
				t.Error(err)
			}

			t.Log("Exif IFD (Exif Private Tag)")
			t.Log(exif)
			//Just an attempt at understanding these crazy MakerNotes..
			for i := range exif.FIA {
				if exif.FIA[i].Tag == MakerNote {
					makernote, err := ExtractMetaData(bytes.NewReader(*exif.FIAvals[i].ascii), 0, 0)
					if err != nil || makernote.Count == 0 {
						t.Error(err)
					}

					//t.Log("Really stupid propietary makernote structure.")
					//t.Log(makernote)
					//for _,v := range makernote.FIA {
					//	t.Logf("%+v\n",v)
					//}
				}
			}
		}

		if fia.Tag == DNGPrivateData {
			dng, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
			if err != nil {
				t.Error(err)
			}

			t.Log("DNG IFD (RAW metadata)")
			t.Log(dng)

			for _, v := range dng.FIA {
				t.Logf("%+v\n", v)
			}

			for i := range dng.FIA {
				if dng.FIA[i].Tag == IDC_IFD {
					idc, err := ExtractMetaData(testARW, int64(dng.FIA[i].Offset), 0)
					if err != nil {
						t.Error(err)
					}

					t.Log("IDC IFD (RAW metadata)")
					t.Log(idc)

					for _, v := range idc.FIA {
						t.Logf("%+v\n", v)
					}
				}
			}
		}
	}
	first, err := ExtractMetaData(testARW, int64(meta.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	t.Log("First IFD for thumbnail data")
	t.Log(first)
}

func TestNestedHeader(t *testing.T) {
	samplename := samples[raw14][0]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}

	meta, err := ExtractMetaData(testARW, 52082, 0)
	if err != nil {
		t.Error(err)
	}
	for _, v := range meta.FIA {
		t.Logf("%+v\n", v)
	}

	var sr2offset uint32
	var sr2length uint32
	var sr2key [4]byte
	for i := range meta.FIA {
		if meta.FIA[i].Tag == SR2SubIFDOffset {
			offset := meta.FIA[i].Offset
			sr2offset = offset
		}
		if meta.FIA[i].Tag == SR2SubIFDLength {
			sr2length = meta.FIA[i].Offset
		}
		if meta.FIA[i].Tag == SR2SubIFDKey {
			key := meta.FIA[i].Offset*0x0edd + 1
			sr2key[3] = byte((key >> 24) & 0xff)
			sr2key[2] = byte((key >> 16) & 0xff)
			sr2key[1] = byte((key >> 8) & 0xff)
			sr2key[0] = byte((key) & 0xff)
		}
	}

	t.Logf("SR2len: %v SR2off: %v SR2key: %v\n", sr2length, sr2offset, sr2key)

	buf := DecryptSR2(testARW, sr2offset, sr2length)
	f, _ := ioutil.TempFile(os.TempDir(), "SR2")
	f.Write(buf)

	br := bytes.NewReader(buf)

	meta, err = ExtractMetaData(br, 0, 0)
	if err != nil {
		t.Error(err)
	}
	t.Log(meta)

	for _, v := range meta.FIA {
		t.Logf("%+v\n", v)
	}
}

func TestJPEGDecode(t *testing.T) {
	testARW, err := os.Open("2.ARW")
	if err != nil {
		t.Error(err)
	}
	header, err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW, int64(header.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	var jpegOffset uint32
	var jpegLength uint32
	for i := range meta.FIA {
		switch meta.FIA[i].Tag {
		case JPEGInterchangeFormat:
			jpegOffset = meta.FIA[i].Offset
		case JPEGInterchangeFormatLength:
			jpegLength = meta.FIA[i].Offset
		}
	}
	jpg, err := ExtractThumbnail(testARW, jpegOffset, jpegLength)
	if err != nil {
		t.Error(err)
	}
	reader := bytes.NewReader(jpg)
	img, err := jpeg.Decode(reader)
	if err != nil {
		t.Error(err)
	}

	out, err := os.Create(fmt.Sprint(time.Now().Unix(), "reencoded", ".jpg"))
	if err != nil {
		t.Error(err)
	}
	jpeg.Encode(out, img, nil)
}

func TestJPEG(t *testing.T) {
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}

	header, err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW, int64(header.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	var jpegOffset uint32
	var jpegLength uint32
	for i := range meta.FIA {
		switch meta.FIA[i].Tag {
		case JPEGInterchangeFormat:
			jpegOffset = meta.FIA[i].Offset
		case JPEGInterchangeFormatLength:
			jpegLength = meta.FIA[i].Offset
		}
	}

	t.Log("JPEG start:", jpegOffset, " JPEG size:", jpegLength)
	jpg := make([]byte, jpegLength)
	testARW.ReadAt(jpg, int64(jpegOffset))
	out, err := os.Create(fmt.Sprint(time.Now().Unix(), "raw", ".jpg"))
	out.Write(jpg)
}
