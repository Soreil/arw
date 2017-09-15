package arw

import (
	"io"
	"errors"
	"encoding/binary"
	"fmt"
	"strings"
	"strconv"
)

type metadata struct {
	TIFFHeader
	zeroth EXIFIFD
}

type TIFFHeader struct {
	ByteOrder uint16
	FortyTwo uint16
	Offset uint32
}

type EXIFIFD struct {
	Count uint16
	FIA []IFDFIA
	FIAvals []FIAval
	Offset uint32
}

func (e EXIFIFD) String() string {
	var result []string
	result = append( result,fmt.Sprintf("Count: %v",e.Count))
	for _,fia := range e.FIA {
		result = append(result,fmt.Sprintf("%+v",fia))
	}
	result = append(result,fmt.Sprintf("Offset to next EXIFIFD: %v",e.Offset))
	return strings.Join(result,"\n")
}

type FIAval struct {
	IFDtype
	ascii *[]byte
	short *[]uint16
	long *[]uint32
	slong *[]int32
	longlong *[]uint64
	slonglong *[]int64
}

func (f FIAval) String() string {
	var val string
	switch f.IFDtype {
	case 1,2,7:
		val = fmt.Sprint(string(*f.ascii))
	case 3:
		val = fmt.Sprint(*f.short)
	case 4:
		val = fmt.Sprint(*f.long)
	case 9:
		val = fmt.Sprint(*f.slong)
	case 5,10:
		val = ""
	}

	return f.IFDtype.String()+": "+val
}

type IFDtag uint16

func (i IFDtag) String() string {
	var r string
	switch i {
	case 254:
		r = "Exif.Image.NewSubfileType"
	case 259:
		r= "Exif.Image.Compression"
	case 270:
		r = "Exif.Image.ImageDescription"
	case 271:
		r = "Exif.Image.Make"
	case 272:
		r = "Exif.Image.Model"
	case 274:
		r = "Exif.Image.Orientation"
	case 282:
		r = "Exif.Image.XResolution"
	case 283:
		r = "Exif.Image.YResolution"
	case 296:
		r = "Exif.Image.ResolutionUnit"
	case 305:
		r = "Exif.Image.Software"
	case 306:
		r = "Exif.Image.DateTime"
	case 330:
		r = "Exif.Image.SubIFDs"
	case 513:
		r = "Exif.Image.JPEGInterchangeFormat"
	case 514:
		r = "Exif.Image.JPEGInterchangeFormatLength"
	case 531:
		r = "Exif.Image.YCbCrPositioning"
	case 34665:
		r = "Exif.Image.ExifTag"
	case 34853:
		r = "Exif.Image.GPSTag"
	case 40965:
		r = "Exif.Photo.InteroperabilityTag"
	case 50341:
		r = "Exif.Image.PrintImageMatching"
	case 50740:
		r = "Exif.Image.DNGPrivateData"
	default:
		r = strconv.Itoa(int(i))
	}
	return r
}

type IFDtype uint16

//Length of bytes
func (i IFDtype) Len() int {
	switch i {
	case 1,2,7:
		return 1
	case 3:
		return 2
	case 4,9:
		return 4
	case 5,10:
		return 8
	default:
		panic("Unknown IFDtype")
	}
}

func (i IFDtype) String() string {
	var r string
	switch i {
	case 1:
		r= "BYTE"
	case 2:
		r= "ASCII"
	case 3:
		r= "SHORT"
	case 4:
		r= "LONG"
	case 5:
		r= "RATIONAL"
	case 7:
		r= "UNDEFINED"
	case 9:
		r= "SLONG"
	case 10:
		r= "SRATIONAL"
	default:
		panic("Unknown IFDtype: "+strconv.Itoa(int(i)))
		}
	return r
}

//IFD Field Interoperability Array
type IFDFIA struct {
	Tag IFDtag
	Type IFDtype
	Count uint32
	Offset uint32
}

//Anyone who thinks I'm switching byte order mid program is sorely mistaken.
var b binary.ByteOrder

func extractMetaData(r io.ReadSeeker) (m metadata,err error) {
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

	binary.Read(r,b,&m.zeroth.Count)
	m.zeroth.FIA = make([]IFDFIA,int(m.zeroth.Count))
	binary.Read(r,b,&m.zeroth.FIA)
	binary.Read(r,b,&m.zeroth.Offset)

	m.zeroth.FIAvals = make([]FIAval,len(m.zeroth.FIA))
	for n,interop := range m.zeroth.FIA {
		m.zeroth.FIAvals[n].IFDtype = interop.Type

		//Offset field is actually the value
		if uint32(interop.Type.Len())*interop.Count <= 4 {
			switch interop.Type {
			case 1,2,7:
				values := make([]byte,interop.Count)
				for i := range values {
					values[i] = byte(((interop.Offset << uint32(8*i)) & 0xff000000)>>24)
				}
				m.zeroth.FIAvals[n].ascii = &values
			case 3:
				values := make([]uint16,interop.Count)
				for i := range values {
					values[i] = uint16(((interop.Offset << uint32(16*i)) & 0xffff0000)>>16)
				}
				m.zeroth.FIAvals[n].short = &values
			case 4:
				values := []uint32{interop.Count}
				m.zeroth.FIAvals[n].long = &values
			case 9:
				values := []int32{int32(interop.Count)}
				m.zeroth.FIAvals[n].slong = &values
			}
		} else {
			r.Seek(int64(interop.Offset),0)
			switch interop.Type {
			case 1, 2, 7:
				values := make([]byte, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].ascii = &values
			case 3:
				values := make([]uint16, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].short = &values
			case 4:
				values := make([]uint32, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].long = &values
			case 9:
				values := make([]int32, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].slong = &values
			case 5:
				values := make([]uint64, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].longlong = &values
			case 10:
				values := make([]int64, interop.Count)
				binary.Read(r,b,&values)
				m.zeroth.FIAvals[n].slonglong = &values
			}

		}
	}

	return
}

//TODO(sjon): spec claims I should handle NULLs for ASCII
func readByte(r io.Reader) byte{
	var bt byte
	binary.Read(r,b,&bt)
	return bt
}

func readUint16(r io.Reader) uint16{
	var short uint16
	binary.Read(r,b,&short)
	return short
}

func readUint32(r io.Reader) uint32{
	var long uint32
	binary.Read(r,b,&long)
	return long
}

//Used for fixpoint of 32 bit numerator and denominator
func readUint64(r io.Reader) uint64{
	var longlong uint64
	binary.Read(r,b,&longlong)
	return longlong
}

func readInt32(r io.Reader) int32{
	var long int32
	binary.Read(r,b,&long)
	return long
}

//Used for fixpoint of 32 bit numerator and denominator
func readInt64(r io.Reader) int64{
	var longlong int64
	binary.Read(r,b,&longlong)
	return longlong
}
