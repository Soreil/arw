package arw

import (
	"image/jpeg"
	"os"
	"testing"
	"time"
	"fmt"
)

const testFileLocation = "samples"

func TestMetadata(t *testing.T) {
	os.Chdir(testFileLocation)
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}

	meta, err := ExtractMetaData(testARW)
	if err != nil {
		t.Error(err)
	}

	t.Log(meta)

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

	out,err := os.Create(fmt.Sprint(time.Now().Unix(),".jpg"))
	if err != nil {
		t.Error(err)
	}

	jpeg.Encode(out,jpg,nil)
}