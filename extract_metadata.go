package arw

import (
	"image"
	"io"
	"log"
)

type rawDetails struct {
	width         uint16
	height        uint16
	bitDepth      uint16
	rawType       sonyRawFile
	offset        uint32
	stride        uint32
	length        uint32
	blackLevel    [4]uint16
	WhiteBalance  [4]int16
	gammaCurve    [5]uint16
	crop          image.Rectangle
	cfaPattern    [4]uint8 //TODO(sjon): This might not always be 4 bytes is my suspicion. We currently take from the offset
	cfaPatternDim [2]uint16
	aperture      float32
	shutter       float32
	iso           uint16
	focalLength   float32
	lensModel     string
}

func extractDetails(rs io.ReadSeeker) (rawDetails, error) {
	var rw rawDetails

	header, err := ParseHeader(rs)
	meta, err := ExtractMetaData(rs, int64(header.Offset), 0)
	if err != nil {
		return rw, err
	}

	for _, fia := range meta.FIA {
		if fia.Tag == SubIFDs {
			rawIFD, err := ExtractMetaData(rs, int64(fia.Offset), 0)
			if err != nil {
				return rw, err
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
					rw.stride = v.Offset //TODO(sjon): Uncompressed RAW files are 2 bytes per pixel whereas CRAW is 1 byte per pixel, this shouldn't be set here! current behaviour is for CRAW, add a divide by 2 for RAW
				case StripByteCounts:
					rw.length = v.Offset
				case SonyCurve:
					curve := *rawIFD.FIAvals[i].short
					copy(rw.gammaCurve[:4], curve)
					rw.gammaCurve[4] = 0x3fff
				case BlackLevel2:
					black := *rawIFD.FIAvals[i].short
					copy(rw.blackLevel[:], black)
				case WB_RGGBLevels:
					balance := *rawIFD.FIAvals[i].sshort
					copy(rw.WhiteBalance[:], balance)
				case DefaultCropSize:
				case CFAPattern2:
					rw.cfaPattern[0] = uint8((v.Offset & 0x000000ff) >> 0)
					rw.cfaPattern[1] = uint8((v.Offset & 0x0000ff00) >> 8)
					rw.cfaPattern[2] = uint8((v.Offset & 0x00ff0000) >> 16)
					rw.cfaPattern[3] = uint8((v.Offset & 0xff000000) >> 24)
				case CFARepeatPatternDim:
					rw.cfaPatternDim[0] = uint16((v.Offset & 0x0000ffff) >> 0)
					rw.cfaPatternDim[1] = uint16((v.Offset & 0xffff0000) >> 16)
				}
			}
		}

		if fia.Tag == ExifTag {
			exif, err := ExtractMetaData(rs, int64(fia.Offset), 0)
			if err != nil {
				return rw, err
			}
			for i, v := range exif.FIA {
				switch v.Tag {
				case ExposureTime:
					rw.shutter = (*exif.FIAvals[i].rat)[0]
				case FNumber:
					rw.aperture = (*exif.FIAvals[i].rat)[0]
				case ISOSpeedRatings:
					rw.iso = uint16((v.Offset & 0x0000ffff) >> 0)
				case FocalLength:
					rw.focalLength = (*exif.FIAvals[i].rat)[0]
				case LensModel:
					rw.lensModel = string(*exif.FIAvals[i].ascii)
				}
			}

		}

		//if fia.Tag == DNGPrivateData {
		//	dng, err := ExtractMetaData(rs, int64(fia.Offset), 0)
		//	if err != nil {
		//		return rw, err
		//	}
		//
		//	var sr2offset uint32
		//	var sr2length uint32
		//	var sr2key [4]byte
		//
		//	for i := range dng.FIA {
		//		if dng.FIA[i].Tag == SR2SubIFDOffset {
		//			offset := dng.FIA[i].Offset
		//			sr2offset = offset
		//		}
		//		if dng.FIA[i].Tag == SR2SubIFDLength {
		//			sr2length = dng.FIA[i].Offset
		//		}
		//		if dng.FIA[i].Tag == SR2SubIFDKey {
		//			key := dng.FIA[i].Offset*0x0edd + 1
		//			sr2key[3] = byte((key >> 24) & 0xff)
		//			sr2key[2] = byte((key >> 16) & 0xff)
		//			sr2key[1] = byte((key >> 8) & 0xff)
		//			sr2key[0] = byte((key) & 0xff)
		//		}
		//	}
		//
		//	buf := DecryptSR2(rs, sr2offset, sr2length)
		//	br := bytes.NewReader(buf)
		//
		//	sr2, err := ExtractMetaData(br, 0, 0)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//
		//	for i, v := range sr2.FIA {
		//		switch v.Tag {
		//		case BlackLevel2:
		//			black := *sr2.FIAvals[i].short
		//			copy(rw.blackLevel[:], black)
		//		case WB_RGGBLevels:
		//			balance := *sr2.FIAvals[i].sshort
		//			copy(rw.WhiteBalance[:], balance)
		//		}
		//	}
		//}
	}

	log.Printf("%+v\n", rw)
	return rw, nil
}
