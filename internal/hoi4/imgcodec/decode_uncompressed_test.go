package imgcodec

import (
	"encoding/binary"
	"testing"
)

func uncompressedDDS(width, height, bits int, r, g, b, a uint32, body []byte) []byte {
	out := make([]byte, ddsHeaderSize)
	copy(out[0:4], "DDS ")
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(out[off:off+4], v) }
	put(4, 124)
	put(8, ddsdCaps|ddsdHeight|ddsdWidth|ddsdPixelFormat)
	put(12, uint32(height))
	put(16, uint32(width))
	put(76, 32)
	flags := uint32(ddpfRGB)
	if a != 0 {
		flags |= ddpfAlphaPixels
	}
	put(80, flags)
	put(88, uint32(bits))
	put(92, r)
	put(96, g)
	put(100, b)
	put(104, a)
	put(108, ddscapsTexture)
	return append(out, body...)
}

func TestDecodeDDSReadsA1R5G5B5(t *testing.T) {

	body := make([]byte, 4)
	binary.LittleEndian.PutUint16(body[0:2], 0x8000|(31<<10))
	binary.LittleEndian.PutUint16(body[2:4], 0x8000|31)
	pix, w, h, ok := DecodeDDS(uncompressedDDS(2, 1, 16, 0x7C00, 0x03E0, 0x001F, 0x8000, body))
	if !ok || w != 2 || h != 1 {
		t.Fatalf("decode failed: ok=%v %dx%d", ok, w, h)
	}

	if got := [4]byte{pix[0], pix[1], pix[2], pix[3]}; got != [4]byte{255, 0, 0, 255} {
		t.Errorf("first pixel = %v, want opaque red", got)
	}
	if got := [4]byte{pix[4], pix[5], pix[6], pix[7]}; got != [4]byte{0, 0, 255, 255} {
		t.Errorf("second pixel = %v, want opaque blue", got)
	}
}

func TestDecodeDDSFollowsChannelMasksNotByteOrder(t *testing.T) {

	body := []byte{0x10, 0x20, 0x30, 0x00}
	pix, _, _, ok := DecodeDDS(uncompressedDDS(1, 1, 32, 0x00FF0000, 0x0000FF00, 0x000000FF, 0, body))
	if !ok {
		t.Fatal("32-bit masked DDS was rejected")
	}
	if got := [4]byte{pix[0], pix[1], pix[2], pix[3]}; got != [4]byte{0x30, 0x20, 0x10, 255} {
		t.Errorf("pixel = %v, want the masks applied and full alpha", got)
	}
}

func TestDecodeDDSLeavesCompressedFilesToTheBlockDecoder(t *testing.T) {
	header := uncompressedDDS(4, 4, 32, 0, 0, 0, 0, make([]byte, 64))
	binary.LittleEndian.PutUint32(header[80:84], ddpfFourCC)
	copy(header[84:88], "DXT1")
	if _, _, _, ok := decodeUncompressedDDS(header); ok {
		t.Error("a FourCC file must not be read as raw pixels")
	}
}

func TestDecodeDDSRejectsTruncatedPixelData(t *testing.T) {
	if _, _, _, ok := DecodeDDS(uncompressedDDS(8, 8, 16, 0x7C00, 0x03E0, 0x001F, 0x8000, make([]byte, 4))); ok {
		t.Error("a file shorter than its own dimensions must be refused")
	}
}
