package arw

import (
	"io"
	"errors"
	"encoding/binary"
)

type metadata struct {
	TIFFHeader
	EXIFIFD
}

type TIFFHeader struct {
	ByteOrder uint16
	FortyTwo uint16
	Offset uint32
}

type EXIFIFD struct {
	Count int16
	FIA IFDFIA
	Offset uint32
}

//IFD Field Interoperability Array
type IFDFIA struct {
	Tag uint16
	Type uint16
	Count uint32
	Offset uint32
}

func extractMetaData(r io.ReadSeeker) (m metadata,err error) {
	var b binary.ByteOrder
	endian := make([]byte,2)
	r.Read(endian)
	switch string(endian) {
	case "II":
		b = binary.LittleEndian
	case "MM":
		b = binary.BigEndian
	default:
		return m,errors.New("failed to determine endianness, unsupposed reader")
	}
	r.Seek(0,0)

	var header TIFFHeader
	binary.Read(r,b,&header)
	r.Seek(int64(header.Offset),0)
	m.TIFFHeader = header

	var zeroth EXIFIFD
	binary.Read(r,b,&zeroth)
	m.EXIFIFD = zeroth

	return
}
