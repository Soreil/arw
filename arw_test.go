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

		for _, v := range rawIFD.FIA {
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

	for i,pix := range data {
		var r,g,b uint16

		if (i %int(rw.stride))% 2 == 0 {
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

		img.Set(i%int(rw.width),i/int(rw.width),color.RGBA64{r*4,g*4,b*4,0xffff})
	}

	os.Chdir("experiments")
	f,err := os.Create("A7R3"+fmt.Sprint(time.Now().Unix())+".png")
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 100; i++ {
		t.Log(img.At(i,0))
	}

	png.Encode(f,img)
}

func TestDecodeBayer(t *testing.T) {
	samplename := samples[craw][0]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}

	//assuming 12 bit file, 36 bits per pixel
	//we will need 2 lines for both RG and GB values
	const offset = 0xD5200
	const width = 6048
	const height = 4024
	const rowsperstrip = height
	const stripbytecount = 0x1735B00
	buf := make([]byte, stripbytecount)
	_, err = testARW.ReadAt(buf, offset)
	if err != nil {
		panic(err)
	}

	t.Log(readCrawBlock(buf[0:width]))
	t.Log(readCrawBlock(buf[0:width]).Decompress())

	//img := image.NewNRGBA64(image.Rect(0,0,width,height))
	//
	//os.Chdir("experiments")
	//time := fmt.Sprint(time.Now().Unix())
	//fjpg,_ := os.Create("F828"+time+".jpg")
	//fpng,_ := os.Create("F828"+time+".png")
	//err = jpeg.Encode(fjpg,img,&jpeg.Options{100})
	//if err != nil {
	//	panic(err)
	//}
	//err = png.Encode(fpng,img)
	//if err != nil {
	//	panic(err)
	//}
}

func TestF828(t *testing.T) {
	os.Chdir(testFileLocation)
	testARW, err := os.Open("F828.SRF.clear")
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

			t.Log()
			t.Log("Exif IFD (Exif Private Tag)")
			for i, v := range exif.FIA {
				if v.Count < 100 {
					t.Logf("%+v\n", exif.FIAvals[i])
				}
				t.Logf("%+v\n", v)
			}
			//Just an attempt at understanding these crazy MakerNotes..
			for i := range exif.FIA {
				if exif.FIA[i].Tag == MakerNote {
					makernote, err := ExtractMetaData(bytes.NewReader(*exif.FIAvals[i].ascii), 0, 0)
					if err != nil || makernote.Count == 0 {
						t.Error(err)
					}
					t.Log()
					t.Log("Makernote")
					t.Log(makernote.Count, makernote.Offset)
					//t.Log(makernote)
					for i, v := range makernote.FIA {
						if v.Count < 1000 {
							t.Logf("%+v\n", exif.FIAvals[i])
						}
						t.Logf("%+v\n", v)
					}
					poked, err := ExtractMetaData(testARW, int64(makernote.Offset), 0)
					if err != nil {
						t.Error(err)
						t.Log(poked)
					}
				}
			}
		}
	}

	first, err := ExtractMetaData(testARW, int64(meta.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	t.Log()
	t.Log("First IFD for thumbnail data")
	t.Log(first)

	for _, v := range first.FIA {
		t.Logf("%+v\n", v)
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
	}

	first, err := ExtractMetaData(testARW, int64(meta.Offset), 0)
	if err != nil {
		t.Error(err)
	}

	t.Log("First IFD for thumbnail data")
	t.Log(first)
}

func TestNestedHeader(t *testing.T) {
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}

	meta, err := ExtractMetaData(testARW, 0xC0DA, 0)
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

func TestArwPoke(t *testing.T) {
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}
	const rawstart = 872960
	const rawend = 25210111
	raw := make([]byte, rawend-rawstart)
	testARW.ReadAt(raw, rawstart)
	f, _ := os.Create("rawoutput.arw")
	f.Write(raw)
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

	t.Log("JPEG start:", jpegOffset, " JPEG size:", jpegLength)
	jpg := make([]byte, jpegLength)
	testARW.ReadAt(jpg, int64(jpegOffset))
	out, err := os.Create(fmt.Sprint(time.Now().Unix(), "raw", ".jpg"))
	out.Write(jpg)
}
