//package arw implements basic support for Exif 2.3 according to CIPA DC-008-2012
package arw

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"image"
	"bytes"
	"image/jpeg"
)

//CIPA DC-008-2012 Table 1
type TIFFHeader struct {
	ByteOrder uint16
	FortyTwo  uint16
	Offset    uint32
}

//CIPA DC-008-2012 Chapter 4.6.2
type EXIFIFD struct {
	Count   uint16
	FIA     []IFDFIA
	FIAvals []FIAval
	Offset  uint32
}

func (e EXIFIFD) String() string {
	var result []string
	result = append(result, fmt.Sprintf("Count: %v", e.Count))
	for i := range e.FIA {
		result = append(result, fmt.Sprintf("%+v", e.FIA[i]))
		result = append(result, fmt.Sprintf("%+v", e.FIAvals[i]))
	}
	result = append(result, fmt.Sprintf("Offset to next EXIFIFD: %v", e.Offset))
	return strings.Join(result, "\n")
}

//CIPA DC-008-2012 Chapter 4.6.2
type FIAval struct {
	IFDtype
	ascii     *[]byte
	short     *[]uint16
	long      *[]uint32
	slong     *[]int32
	longlong  *[]uint64
	slonglong *[]int64
}

func (f FIAval) String() string {
	var val string
	switch f.IFDtype {
	case 1, 2, 7:
		val = fmt.Sprint(string(*f.ascii))
	case 3:
		val = fmt.Sprint(*f.short)
	case 4:
		val = fmt.Sprint(*f.long)
	case 9:
		val = fmt.Sprint(*f.slong)
	case 5:
		for _, long := range *f.longlong {
			val += fmt.Sprintf("%v.%v, ", (long>>32)&0xffffffff, long&0xffffffff)
		}
	case 10:
		for _, long := range *f.slonglong {
			val += fmt.Sprintf("%v.%v, ", (long>>32)&0xffffffff, long&0xffffffff)
		}
	}

	return f.IFDtype.String() + ": " + val
}

//go:generate stringer -type=IFDtag
type IFDtag uint16

//IFDtags mapping taken from http://www.exiv2.org/tags.html
const (
	NewSubFileType              IFDtag = 254
	Compression                 IFDtag = 259
	ImageDescription            IFDtag = 270
	Make                        IFDtag = 271
	Model                       IFDtag = 272
	Orientation                 IFDtag = 274
	XResolution                 IFDtag = 282
	YResolution                 IFDtag = 283
	ResolutionUnit              IFDtag = 296
	Software                    IFDtag = 305
	DateTime                    IFDtag = 306
	SubIFDs                     IFDtag = 330
	JPEGInterchangeFormat       IFDtag = 513
	JPEGInterchangeFormatLength IFDtag = 514
	YCbCrPositioning            IFDtag = 531
	ExifTag                     IFDtag = 34665
	GPSTag                      IFDtag = 34853
	InteroperabilityTag         IFDtag = 40965
	PrintImageMatching          IFDtag = 50341
	DNGPrivateData              IFDtag = 50740
)

//IFD datatype, most datatypes translate in to C datatypes.
type IFDtype uint16

//IFDType length in bytes
func (i IFDtype) Len() int {
	switch i {
	case 1, 2, 7:
		return 1
	case 3:
		return 2
	case 4, 9:
		return 4
	case 5, 10:
		return 8
	default:
		panic("Unknown IFDtype")
	}
}

func (i IFDtype) String() string {
	var r string
	switch i {
	case 1:
		r = "BYTE"
	case 2:
		r = "ASCII"
	case 3:
		r = "SHORT"
	case 4:
		r = "LONG"
	case 5:
		r = "RATIONAL"
	case 7:
		r = "UNDEFINED"
	case 9:
		r = "SLONG"
	case 10:
		r = "SRATIONAL"
	default:
		panic("Unknown IFDtype: " + strconv.Itoa(int(i)))
	}
	return r
}

//IFD Field Interoperability Array
//CIPA DC-008-2012 Chapter 4.6.2
type IFDFIA struct {
	Tag    IFDtag
	Type   IFDtype
	Count  uint32
	Offset uint32
}

//Anyone who thinks I'm switching byte order mid program is sorely mistaken.
var b binary.ByteOrder

//ExtractMetadata will return the first IFD from a TIFF document.
func ExtractMetaData(r io.ReadSeeker) (meta EXIFIFD, err error) {
	endian := make([]byte, 2)
	r.Read(endian)
	switch string(endian) {
	case "II":
		b = binary.LittleEndian
	case "MM":
		b = binary.BigEndian
	default:
		return meta, errors.New("failed to determine endianness")
	}
	r.Seek(0, 0)

	var header TIFFHeader
	binary.Read(r, b, &header)
	r.Seek(int64(header.Offset), 0)

	binary.Read(r, b, &meta.Count)
	meta.FIA = make([]IFDFIA, int(meta.Count))
	binary.Read(r, b, &meta.FIA)
	binary.Read(r, b, &meta.Offset)

	meta.FIAvals = make([]FIAval, len(meta.FIA))
	for n, interop := range meta.FIA {
		meta.FIAvals[n].IFDtype = interop.Type

		//Offset field is actually the value
		if uint32(interop.Type.Len())*interop.Count <= 4 {
			switch interop.Type {
			case 1, 2, 7:
				values := make([]byte, interop.Count)
				for i := range values {
					values[i] = byte(((interop.Offset << uint32(8*i)) & 0xff000000) >> 24)
				}
				meta.FIAvals[n].ascii = &values
			case 3:
				values := make([]uint16, interop.Count)
				for i := range values {
					values[i] = uint16(((interop.Offset << uint32(16*i)) & 0xffff0000) >> 16)
				}
				meta.FIAvals[n].short = &values
			case 4:
				values := []uint32{interop.Count}
				meta.FIAvals[n].long = &values
			case 9:
				values := []int32{int32(interop.Count)}
				meta.FIAvals[n].slong = &values
			}
		} else {
			r.Seek(int64(interop.Offset), 0)
			switch interop.Type {
			case 1, 2, 7:
				values := make([]byte, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].ascii = &values
			case 3:
				values := make([]uint16, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].short = &values
			case 4:
				values := make([]uint32, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].long = &values
			case 9:
				values := make([]int32, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].slong = &values
			case 5:
				values := make([]uint64, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].longlong = &values
			case 10:
				values := make([]int64, interop.Count)
				binary.Read(r, b, &values)
				meta.FIAvals[n].slonglong = &values
			}

		}
	}

	return
}

//ExtractThumbnail extracts an embedded JPEG thumbnail.
//The extracted image.Image has no EXIF data included.
func ExtractThumbnail(r io.ReaderAt,offset uint32,length uint32) (image.Image, error){
	jpegData := make([]byte, length)
	_,err := r.ReadAt(jpegData, int64(offset))
	if err != nil {
		return nil,err
	}
	reader := bytes.NewReader(jpegData)
	jpg, err := jpeg.Decode(reader)
	if err != nil {
		return nil,err
	}
	return jpg,nil
}

//TODO(sjon): spec claims I should handle NULLs for ASCII
func readByte(r io.Reader) byte {
	var bt byte
	binary.Read(r, b, &bt)
	return bt
}

func readUint16(r io.Reader) uint16 {
	var short uint16
	binary.Read(r, b, &short)
	return short
}

func readUint32(r io.Reader) uint32 {
	var long uint32
	binary.Read(r, b, &long)
	return long
}

//Used for fixpoint of 32 bit numerator and denominator
func readUint64(r io.Reader) uint64 {
	var longlong uint64
	binary.Read(r, b, &longlong)
	return longlong
}

func readInt32(r io.Reader) int32 {
	var long int32
	binary.Read(r, b, &long)
	return long
}

//Used for fixpoint of 32 bit numerator and denominator
func readInt64(r io.Reader) int64 {
	var longlong int64
	binary.Read(r, b, &longlong)
	return longlong
}
