package arw

import (
	"io"
	"errors"
)

type metadata struct {
	isLittleEndian bool
	totalSize int64
	thumbnailSize int64
	rawSize int64
	make string
	manufacturer string
}

func extractMetaData(r io.Reader) (metadata,error) {
	var m metadata
	tiffHeader := make([]byte,0,8)
	r.Read(tiffHeader)
	if len(tiffHeader) != 8 {
		return m,errors.New("failed to read TIFF header")
	}
	switch string(tiffHeader[:2]) {
	case "II" :
		m.isLittleEndian = true
	case "MM" :
		m.isLittleEndian = false
	default:
		return m, errors.New("can't determine file endianness")
	}

	return m,nil
}
