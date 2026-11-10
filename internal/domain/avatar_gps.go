package domain

import (
	"bytes"
	"encoding/binary"
)

// AvatarHasLocation reports whether an avatar file carries a GPS position: a
// GPS directory with a latitude or longitude in its EXIF block, or the same
// fields in an XMP packet. Avatars are served to anyone who has the URL, so a
// phone photo would otherwise publish where it was taken.
//
// The file is not decoded as an image. A block that cannot be parsed is treated
// as having no position.
func AvatarHasLocation(b []byte, contentType string) bool {
	switch contentType {
	case AvatarJPEG:
		return jpegHasLocation(b)
	case AvatarPNG:
		return pngHasLocation(b)
	case AvatarWebP:
		return webpHasLocation(b)
	}
	return false
}

var (
	exifHeader = []byte("Exif\x00\x00")
	xmpGPSLat  = []byte("GPSLatitude")
	xmpGPSLong = []byte("GPSLongitude")
)

func xmpHasLocation(b []byte) bool {
	return bytes.Contains(b, xmpGPSLat) || bytes.Contains(b, xmpGPSLong)
}

// jpegHasLocation walks the marker segments before the image data. EXIF and XMP
// both live in APP1 segments.
func jpegHasLocation(b []byte) bool {
	i := 2 // after SOI
	for i+4 <= len(b) {
		if b[i] != 0xff {
			return false
		}
		marker := b[i+1]
		if marker == 0xff { // fill byte
			i++
			continue
		}
		if marker == 0xda || marker == 0xd9 { // image data, or end
			return false
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) { // no length
			i += 2
			continue
		}
		size := int(binary.BigEndian.Uint16(b[i+2 : i+4]))
		if size < 2 || i+2+size > len(b) {
			return false
		}
		payload := b[i+4 : i+2+size]
		if marker == 0xe1 {
			if bytes.HasPrefix(payload, exifHeader) {
				if tiffHasLocation(payload[len(exifHeader):]) {
					return true
				}
			} else if xmpHasLocation(payload) {
				return true
			}
		}
		i += 2 + size
	}
	return false
}

// pngHasLocation walks the chunks. EXIF is the eXIf chunk; XMP is text in iTXt.
func pngHasLocation(b []byte) bool {
	i := 8 // after the signature
	for i+12 <= len(b) {
		size := int(binary.BigEndian.Uint32(b[i : i+4]))
		kind := string(b[i+4 : i+8])
		if size < 0 || i+8+size > len(b) {
			return false
		}
		data := b[i+8 : i+8+size]
		switch kind {
		case "eXIf":
			if tiffHasLocation(data) {
				return true
			}
		case "iTXt", "tEXt":
			if xmpHasLocation(data) {
				return true
			}
		case "IEND":
			return false
		}
		i += 12 + size
	}
	return false
}

// webpHasLocation walks the RIFF chunks. EXIF and XMP are their own chunks.
func webpHasLocation(b []byte) bool {
	i := 12 // after "RIFF" size "WEBP"
	for i+8 <= len(b) {
		kind := string(b[i : i+4])
		size := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		if size < 0 || i+8+size > len(b) {
			return false
		}
		data := b[i+8 : i+8+size]
		switch kind {
		case "EXIF":
			// Some writers keep the JPEG-style prefix in front of the TIFF block.
			if tiffHasLocation(bytes.TrimPrefix(data, exifHeader)) {
				return true
			}
		case "XMP ":
			if xmpHasLocation(data) {
				return true
			}
		}
		i += 8 + size + size%2 // chunks are padded to an even length
	}
	return false
}

const (
	tiffTagGPSInfo   = 0x8825
	gpsTagLatitude   = 0x0002
	gpsTagLongitude  = 0x0004
	tiffEntrySize    = 12
	tiffMaxIFDEntrys = 1024
)

// tiffHasLocation reads an EXIF (TIFF) block: IFD0 points at the GPS directory,
// which has a position when it holds a latitude or a longitude. A GPS directory
// with only a version tag, which some cameras always write, is not a position.
func tiffHasLocation(t []byte) bool {
	if len(t) < 8 {
		return false
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return false
	}
	if order.Uint16(t[2:4]) != 42 {
		return false
	}
	gps, ok := tiffFindTag(t, order, order.Uint32(t[4:8]), tiffTagGPSInfo)
	if !ok {
		return false
	}
	// The entry's value is the offset of the GPS directory.
	gpsIFD := order.Uint32(gps[8:12])
	if _, ok := tiffFindTag(t, order, gpsIFD, gpsTagLatitude); ok {
		return true
	}
	_, ok = tiffFindTag(t, order, gpsIFD, gpsTagLongitude)
	return ok
}

// tiffFindTag returns the 12-byte entry for tag in the directory at offset.
func tiffFindTag(t []byte, order binary.ByteOrder, offset uint32, tag uint16) ([]byte, bool) {
	start := uint64(offset)
	if start+2 > uint64(len(t)) {
		return nil, false
	}
	n := int(order.Uint16(t[start : start+2]))
	if n > tiffMaxIFDEntrys {
		return nil, false
	}
	for i := 0; i < n; i++ {
		at := start + 2 + uint64(i)*tiffEntrySize
		if at+tiffEntrySize > uint64(len(t)) {
			return nil, false
		}
		entry := t[at : at+tiffEntrySize]
		if order.Uint16(entry[:2]) == tag {
			return entry, true
		}
	}
	return nil, false
}
