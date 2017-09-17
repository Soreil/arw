package arw

import (
	"image/jpeg"
	"os"
	"testing"
	"time"
	"fmt"
	"bytes"
)

const testFileLocation = "samples"

func TestMetadata(t *testing.T) {
	os.Chdir(testFileLocation)
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}

	header,err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW,int64(header.Offset),0)
	if err != nil {
		t.Error(err)
	}
	t.Log("0th IFD for primary image data")
	t.Log(meta)

	for _,v := range meta.FIA {
		t.Logf("%+v\n",v)
	}

	for _,fia := range meta.FIA {
		if fia.Tag == SubIFDs {
			next,err := ExtractMetaData(testARW,int64(fia.Offset),0)
			if err != nil {
				t.Error(err)
			}
			t.Log("A subIFD, who knows what we'll find here!")
			t.Log(next)
		}

		if fia.Tag == GPSTag {
			gps, err := ExtractMetaData(testARW,int64(fia.Offset),0)
			if err != nil {
				t.Error(err)
			}

			t.Log("GPS IFD (GPS Info Tag)")
			t.Log(gps)
		}

		if fia.Tag == ExifTag {
			exif, err := ExtractMetaData(testARW,int64(fia.Offset),0)
			if err != nil {
				t.Error(err)
			}

			t.Log("Exif IFD (Exif Private Tag)")
			t.Log(exif)
			////Just an attempt at understanding these crazy MakerNotes..
			//for i := range exif.FIA {
			//	if exif.FIA[i].Tag == MakerNote {
			//		makernote,err := ExtractMetaData(bytes.NewReader(*exif.FIAvals[i].ascii),0,0)
			//		if err != nil {
			//			t.Error(err)
			//		}
			//
			//		t.Log("Really stupid propietary makernote structure.")
			//		t.Log(makernote)
			//	}
			//}
		}
	}

	first, err := ExtractMetaData(testARW,int64(meta.Offset),0)
	if err != nil {
		t.Error(err)
	}

	t.Log("First IFD for thumbnail data")
	t.Log(first)
}

func TestJPEGDecode(t *testing.T) {
	os.Chdir(testFileLocation)
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}
	header,err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW,int64(header.Offset),0)
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
	jpg,err := ExtractThumbnail(testARW,jpegOffset,jpegLength)
	if err != nil {
		t.Error(err)
	}
	reader := bytes.NewReader(jpg)
	img, err := jpeg.Decode(reader)

	out,err := os.Create(fmt.Sprint(time.Now().Unix(),"reencoded",".jpg"))
	if err != nil {
		t.Error(err)
	}

	jpeg.Encode(out,img,nil)
}

func TestJPEG(t *testing.T) {
	os.Chdir(testFileLocation)
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}

	header,err := ParseHeader(testARW)
	meta, err := ExtractMetaData(testARW,int64(header.Offset),0)
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

	jpg := make([]byte,jpegLength)
	testARW.ReadAt(jpg,int64(jpegOffset))
	out,err := os.Create(fmt.Sprint(time.Now().Unix(),"raw",".jpg"))
	out.Write(jpg)
}