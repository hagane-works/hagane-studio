package imgcodec

import "encoding/binary"

const (
	ddsHeaderSize   = 128
	ddsdCaps        = 0x1
	ddsdHeight      = 0x2
	ddsdWidth       = 0x4
	ddsdPitch       = 0x8
	ddsdPixelFormat = 0x1000
	ddsdLinearSize  = 0x80000

	ddpfAlphaPixels = 0x1
	ddpfFourCC      = 0x4
	ddpfRGB         = 0x40

	ddscapsTexture    = 0x1000
	ddscaps2Cubemap   = 0x200
	ddscaps2Volume    = 0x200000
	fourCCDXT5        = 0x35545844
	maxDDSDimension   = 32768
	bytesPerPixelRGBA = 4
)

func EncodeDXT5DDS(pix []byte, w, h int) []byte {
	blocks := EncodeDXT5Blocks(pix, w, h)
	if blocks == nil {
		return nil
	}
	out := make([]byte, ddsHeaderSize, ddsHeaderSize+len(blocks))
	copy(out[0:4], "DDS ")
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(out[off:off+4], v) }
	put(4, 124)
	put(8, ddsdCaps|ddsdHeight|ddsdWidth|ddsdPixelFormat|ddsdLinearSize)
	put(12, uint32(h))
	put(16, uint32(w))
	put(20, uint32((w+3)*4))

	put(76, 32)
	put(80, ddpfFourCC)
	put(84, fourCCDXT5)
	put(88, 32)

	put(108, ddscapsTexture)
	return append(out, blocks...)
}

type ddsMasks struct{ r, g, b, a uint32 }

var ddsRawMode32 = map[ddsMasks]bool{
	{0x00FF0000, 0x0000FF00, 0x000000FF, 0xFF000000}: true,
	{0x000000FF, 0x0000FF00, 0x00FF0000, 0xFF000000}: false,
}

func FastDecodeDDS(data []byte) (pix []byte, w, h int, ok bool) {
	if len(data) < ddsHeaderSize || string(data[0:4]) != "DDS " {
		return nil, 0, 0, false
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(data[off : off+4]) }
	if u32(4) != 124 {
		return nil, 0, 0, false
	}
	dwFlags := u32(8)
	height, width := u32(12), u32(16)
	pitch := u32(20)
	pfFlags := u32(80)
	fourcc := data[84:88]
	bitcount := u32(88)
	masks := ddsMasks{r: u32(92), g: u32(96), b: u32(100), a: u32(104)}
	caps2 := u32(112)

	if fourcc[0] != 0 || fourcc[1] != 0 || fourcc[2] != 0 || fourcc[3] != 0 ||
		pfFlags&ddpfFourCC != 0 {
		return nil, 0, 0, false
	}
	if pfFlags&ddpfRGB == 0 {
		return nil, 0, 0, false
	}
	if caps2&(ddscaps2Cubemap|ddscaps2Volume) != 0 {
		return nil, 0, 0, false
	}
	if !(width > 0 && width <= maxDDSDimension && height > 0 && height <= maxDDSDimension) {
		return nil, 0, 0, false
	}
	if bitcount != 32 {
		return nil, 0, 0, false
	}

	if pfFlags&ddpfAlphaPixels == 0 {
		return nil, 0, 0, false
	}
	bgra, found := ddsRawMode32[masks]
	if !found {
		return nil, 0, 0, false
	}

	if dwFlags&ddsdPitch != 0 && pitch != 0 && pitch != width*bytesPerPixelRGBA {
		return nil, 0, 0, false
	}
	need := int(width) * int(height) * bytesPerPixelRGBA
	if len(data)-ddsHeaderSize < need {
		return nil, 0, 0, false
	}
	src := data[ddsHeaderSize : ddsHeaderSize+need]
	out := make([]byte, need)
	if bgra {
		for i := 0; i < need; i += 4 {
			out[i] = src[i+2]
			out[i+1] = src[i+1]
			out[i+2] = src[i]
			out[i+3] = src[i+3]
		}
	} else {
		copy(out, src)
	}
	return out, int(width), int(height), true
}
