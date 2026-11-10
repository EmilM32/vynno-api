package domain

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// tiffBlock builds a minimal EXIF block. gps selects the GPS directory:
// "" none, "version" only GPSVersionID, "position" latitude and longitude.
func tiffBlock(order binary.ByteOrder, gps string) []byte {
	var b bytes.Buffer
	if order == binary.LittleEndian {
		b.WriteString("II")
	} else {
		b.WriteString("MM")
	}
	w16 := func(v uint16) { _ = binary.Write(&b, order, v) }
	w32 := func(v uint32) { _ = binary.Write(&b, order, v) }
	entry := func(tag, typ uint16, count, value uint32) { w16(tag); w16(typ); w32(count); w32(value) }
	w16(42)
	w32(8) // IFD0 right after the header

	// IFD0: Orientation, and the GPS pointer when asked for.
	ifd0 := uint16(1)
	if gps != "" {
		ifd0 = 2
	}
	gpsOffset := uint32(8 + 2 + int(ifd0)*12 + 4)
	w16(ifd0)
	entry(0x0112, 3, 1, 1)
	if gps != "" {
		entry(tiffTagGPSInfo, 4, 1, gpsOffset)
	}
	w32(0) // no next IFD

	switch gps {
	case "version":
		w16(1)
		entry(0x0000, 1, 4, 0x02030000)
		w32(0)
	case "position":
		w16(3)
		entry(0x0000, 1, 4, 0x02030000)
		entry(gpsTagLatitude, 5, 3, 0)
		entry(gpsTagLongitude, 5, 3, 0)
		w32(0)
	}
	return b.Bytes()
}

func jpegWith(segments ...[]byte) []byte {
	out := []byte{0xff, 0xd8}
	// A JFIF APP0 first, as real files have.
	segments = append([][]byte{jpegSegment(0xe0, []byte("JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00"))}, segments...)
	for _, s := range segments {
		out = append(out, s...)
	}
	return append(out, 0xff, 0xda, 0x00, 0x02, 0x11, 0x22, 0xff, 0xd9)
}

func jpegSegment(marker byte, payload []byte) []byte {
	out := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(out[2:], uint16(len(payload)+2))
	return append(out, payload...)
}

func pngWith(chunks ...[]byte) []byte {
	out := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	out = append(out, pngChunk("IHDR", make([]byte, 13))...)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return append(out, pngChunk("IEND", nil)...)
}

func pngChunk(kind string, data []byte) []byte {
	out := make([]byte, 4, 12+len(data))
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	out = append(out, kind...)
	out = append(out, data...)
	return append(out, 0, 0, 0, 0) // the CRC is not checked
}

func webpWith(chunks ...[]byte) []byte {
	body := []byte("WEBP")
	body = append(body, webpChunk("VP8X", make([]byte, 10))...)
	for _, c := range chunks {
		body = append(body, c...)
	}
	out := []byte("RIFF\x00\x00\x00\x00")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(body)))
	return append(out, body...)
}

func webpChunk(kind string, data []byte) []byte {
	out := append([]byte(kind), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func TestAvatarHasLocation(t *testing.T) {
	le, be := binary.ByteOrder(binary.LittleEndian), binary.ByteOrder(binary.BigEndian)
	exif := func(order binary.ByteOrder, gps string) []byte {
		return append([]byte("Exif\x00\x00"), tiffBlock(order, gps)...)
	}
	xmp := []byte(`http://ns.adobe.com/xap/1.0/` + "\x00" + `<x:xmpmeta><rdf:Description exif:GPSLatitude="52,13.8N"/></x:xmpmeta>`)
	plainXMP := []byte(`http://ns.adobe.com/xap/1.0/` + "\x00" + `<x:xmpmeta><rdf:Description dc:title="me"/></x:xmpmeta>`)

	for _, tc := range []struct {
		name string
		file []byte
		ct   string
		want bool
	}{
		{"jpeg without metadata", jpegWith(), AvatarJPEG, false},
		{"jpeg exif, no gps", jpegWith(jpegSegment(0xe1, exif(le, ""))), AvatarJPEG, false},
		{"jpeg gps version only", jpegWith(jpegSegment(0xe1, exif(be, "version"))), AvatarJPEG, false},
		{"jpeg gps position, little endian", jpegWith(jpegSegment(0xe1, exif(le, "position"))), AvatarJPEG, true},
		{"jpeg gps position, big endian", jpegWith(jpegSegment(0xe1, exif(be, "position"))), AvatarJPEG, true},
		{"jpeg gps in a later APP1", jpegWith(jpegSegment(0xe1, plainXMP), jpegSegment(0xe1, exif(le, "position"))), AvatarJPEG, true},
		{"jpeg xmp position", jpegWith(jpegSegment(0xe1, xmp)), AvatarJPEG, true},
		{"jpeg xmp without position", jpegWith(jpegSegment(0xe1, plainXMP)), AvatarJPEG, false},
		{"png without metadata", pngWith(), AvatarPNG, false},
		{"png exif position", pngWith(pngChunk("eXIf", tiffBlock(be, "position"))), AvatarPNG, true},
		{"png exif version only", pngWith(pngChunk("eXIf", tiffBlock(le, "version"))), AvatarPNG, false},
		{"png xmp position", pngWith(pngChunk("iTXt", append([]byte("XML:com.adobe.xmp\x00\x00\x00\x00\x00"), xmp...))), AvatarPNG, true},
		{"webp without metadata", webpWith(), AvatarWebP, false},
		{"webp exif position", webpWith(webpChunk("EXIF", tiffBlock(le, "position"))), AvatarWebP, true},
		{"webp exif position, odd chunk before", webpWith(webpChunk("ICCP", []byte{1, 2, 3}), webpChunk("EXIF", tiffBlock(be, "position"))), AvatarWebP, true},
		{"webp exif with jpeg prefix", webpWith(webpChunk("EXIF", exif(le, "position"))), AvatarWebP, true},
		{"webp exif, no gps", webpWith(webpChunk("EXIF", tiffBlock(le, ""))), AvatarWebP, false},
		{"webp xmp position", webpWith(webpChunk("XMP ", xmp)), AvatarWebP, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ct, err := DetectAvatarContentType(tc.file)
			if err != nil || ct != tc.ct {
				t.Fatalf("fixture is not a %s: %q, %v", tc.ct, ct, err)
			}
			if got := AvatarHasLocation(tc.file, ct); got != tc.want {
				t.Fatalf("AvatarHasLocation = %v, want %v", got, tc.want)
			}
		})
	}
}

// Truncated and hostile metadata must not panic or loop; it reads as no position.
func TestAvatarHasLocationSurvivesBrokenFiles(t *testing.T) {
	good := jpegWith(jpegSegment(0xe1, append([]byte("Exif\x00\x00"), tiffBlock(binary.LittleEndian, "position")...)))
	for n := 0; n <= len(good); n++ {
		AvatarHasLocation(good[:n], AvatarJPEG)
	}
	png := pngWith(pngChunk("eXIf", tiffBlock(binary.BigEndian, "position")))
	for n := 0; n <= len(png); n++ {
		AvatarHasLocation(png[:n], AvatarPNG)
	}
	webp := webpWith(webpChunk("EXIF", tiffBlock(binary.LittleEndian, "position")))
	for n := 0; n <= len(webp); n++ {
		AvatarHasLocation(webp[:n], AvatarWebP)
	}

	// Offsets and lengths that point far outside the block.
	huge := tiffBlock(binary.LittleEndian, "position")
	binary.LittleEndian.PutUint32(huge[4:], 0xfffffff0)
	if tiffHasLocation(huge) {
		t.Fatal("IFD offset outside the block was followed")
	}
	bigPNG := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0xff, 0xff, 0xff, 0xff, 'e', 'X', 'I', 'f', 0, 0, 0, 0}
	if AvatarHasLocation(bigPNG, AvatarPNG) {
		t.Fatal("chunk length past the end was followed")
	}
	bigWebP := []byte("RIFF\x10\x00\x00\x00WEBPEXIF\xff\xff\xff\xff")
	if AvatarHasLocation(bigWebP, AvatarWebP) {
		t.Fatal("chunk length past the end was followed")
	}
}

func FuzzAvatarHasLocation(f *testing.F) {
	f.Add(jpegWith(jpegSegment(0xe1, append([]byte("Exif\x00\x00"), tiffBlock(binary.LittleEndian, "position")...))), 0)
	f.Add(pngWith(pngChunk("eXIf", tiffBlock(binary.BigEndian, "position"))), 1)
	f.Add(webpWith(webpChunk("EXIF", tiffBlock(binary.LittleEndian, "position"))), 2)
	types := []string{AvatarJPEG, AvatarPNG, AvatarWebP}
	f.Fuzz(func(t *testing.T, data []byte, kind int) {
		if kind < 0 {
			kind = -kind
		}
		AvatarHasLocation(data, types[kind%len(types)])
	})
}
