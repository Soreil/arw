package arw

import (
	"io"
	"errors"
	"encoding/binary"
)

type metadata struct {
	isLittleEndian bool
	zerothOffset uint32
}

func extractMetaData(r io.ReadSeeker) (metadata,error) {
	var m metadata

	var encoding [2]byte
	if err := binary.Read(r,binary.LittleEndian,&encoding); err != nil {
		return m,err
	}

	switch encoding {
	case [2]byte{'I','I'} :
		m.isLittleEndian = true
	case [2]byte{'M','M'} :
		m.isLittleEndian = false
	default:
		return m, errors.New("can't determine file endianness")
	}

	r.Seek(2,1)

	var offset uint32
	if err := binary.Read(r,binary.LittleEndian,&offset); err != nil {
		return m,err
	}
	m.zerothOffset = offset
	r.Seek(int64(offset),0)


	return m,nil
}
