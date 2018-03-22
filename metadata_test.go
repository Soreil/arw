package arw

import (
	"bytes"
	"os"
	"testing"
)

func TestMetadata(t *testing.T) {
	samplename := samples[raw14][1]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}

	header, err := ParseHeader(testARW)
	if err != nil {
		t.Error(err)
	}

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
			//for i := range exif.FIA {
			//	if exif.FIA[i].Tag == MakerNote {
			//		makernote, err := ExtractMetaData(bytes.NewReader(*exif.FIAvals[i].ascii), 0, 0)
			//		if err != nil || makernote.Count == 0 {
			//			t.Error(err)
			//		}
			//
			//		t.Log("Really stupid propietary makernote structure.")
			//		t.Log(makernote)
			//		for _,v := range makernote.FIA {
			//			t.Logf("%+v\n",v)
			//		}
			//	}
			//}
		}

		if fia.Tag == DNGPrivateData {
			dng, err := ExtractMetaData(testARW, int64(fia.Offset), 0)
			if err != nil {
				t.Error(err)
			}

			t.Log("DNG IFD (RAW metadata)")
			t.Log(dng)

			for _, v := range dng.FIA {
				t.Logf("%+v\n", v)
			}

			var sr2offset uint32
			var sr2length uint32
			var sr2key [4]byte

			for i := range dng.FIA {
				if dng.FIA[i].Tag == IDC_IFD {
					idc, err := ExtractMetaData(testARW, int64(dng.FIA[i].Offset), 0)
					if err != nil {
						t.Error(err)
					}

					t.Log("IDC IFD (RAW metadata)")
					t.Log(idc)

					for _, v := range idc.FIA {
						t.Logf("%+v\n", v)
					}
				}

				if dng.FIA[i].Tag == SR2SubIFDOffset {
					offset := dng.FIA[i].Offset
					sr2offset = offset
				}
				if dng.FIA[i].Tag == SR2SubIFDLength {
					sr2length = dng.FIA[i].Offset
				}
				if dng.FIA[i].Tag == SR2SubIFDKey {
					key := dng.FIA[i].Offset*0x0edd + 1
					sr2key[3] = byte((key >> 24) & 0xff)
					sr2key[2] = byte((key >> 16) & 0xff)
					sr2key[1] = byte((key >> 8) & 0xff)
					sr2key[0] = byte((key) & 0xff)
				}
			}
			buf := DecryptSR2(testARW, sr2offset, sr2length)
			br := bytes.NewReader(buf)

			sr2, err := ExtractMetaData(br, 0, 0)
			if err != nil {
				t.Error(err)
			}
			t.Logf("SR2len: %v SR2off: %v SR2key: %v\n", sr2length, sr2offset, sr2key)
			t.Log(sr2)

			for _, v := range sr2.FIA {
				t.Logf("%+v\n", v)
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
	samplename := samples[raw14][0]
	testARW, err := os.Open(samplename + ".ARW")
	if err != nil {
		t.Error(err)
	}

	meta, err := ExtractMetaData(testARW, 52082, 0)
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
