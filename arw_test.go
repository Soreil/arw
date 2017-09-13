package arw

import (
	"testing"
	"os"
)

const testFileLocation = "samples"

func TestMetadata(t *testing.T) {
	os.Chdir(testFileLocation)
	testJPG, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}
	testARW, err := os.Open("1.ARW")
	if err != nil {
		t.Error(err)
	}
	meta,err := extractMetaData(testJPG)
	if err != nil {
		t.Error(meta)
	}

	t.Logf("JPEG: %+v\n", meta)

	meta,err = extractMetaData(testARW)
	if err != nil {
		t.Error(meta)
	}

	t.Logf("ARW: %+v\n", meta)
}
