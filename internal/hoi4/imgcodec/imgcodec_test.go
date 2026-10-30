package imgcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func readBlob(t *testing.T, path string) (pix []byte, w, h int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(raw) < 12 || string(raw[0:4]) != "RGBA" {
		t.Fatalf("%s: bad blob magic", path)
	}
	w = int(binary.LittleEndian.Uint32(raw[4:8]))
	h = int(binary.LittleEndian.Uint32(raw[8:12]))
	return raw[12:], w, h
}

func TestEncodeDXT5DDSMatchesPillowGoldens(t *testing.T) {
	blobs, err := filepath.Glob("testdata/enc/*.rgba")
	if err != nil || len(blobs) == 0 {
		t.Fatalf("no encode goldens found: %v", err)
	}
	for _, blob := range blobs {
		name := filepath.Base(blob[:len(blob)-len(".rgba")])
		t.Run(name, func(t *testing.T) {
			pix, w, h := readBlob(t, blob)
			want, err := os.ReadFile("testdata/enc/" + name + ".dds")
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			got := EncodeDXT5DDS(pix, w, h)
			if got == nil {
				t.Fatal("EncodeDXT5DDS returned nil")
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%dx%d: %d bytes vs golden %d bytes; first diff at %d",
					w, h, len(got), len(want), firstDiff(got, want))
			}
		})
	}
}

func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

func TestFastDecodeDDSMatchesPillowGoldens(t *testing.T) {
	files, err := filepath.Glob("testdata/dec/*.dds")
	if err != nil || len(files) == 0 {
		t.Fatalf("no decode goldens found: %v", err)
	}
	accepted, rejected := 0, 0
	for _, f := range files {
		name := filepath.Base(f[:len(f)-len(".dds")])
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read %s: %v", f, err)
			}
			gotPix, gotW, gotH, ok := FastDecodeDDS(data)
			wantPath := "testdata/dec/" + name + ".rgba"
			if _, statErr := os.Stat(wantPath); statErr != nil {

				if ok {
					t.Fatalf("accepted an input the PIL fast path rejects "+
						"(%dx%d) — Go would handle a file with different Pillow semantics",
						gotW, gotH)
				}
				rejected++
				return
			}
			if !ok {
				t.Fatal("rejected an input the PIL fast path accepts")
			}
			wantPix, wantW, wantH := readBlob(t, wantPath)
			if gotW != wantW || gotH != wantH {
				t.Fatalf("dims %dx%d, want %dx%d", gotW, gotH, wantW, wantH)
			}
			if !bytes.Equal(gotPix, wantPix) {
				t.Errorf("pixels differ; first diff at %d", firstDiff(gotPix, wantPix))
			}
			accepted++
		})
	}
	t.Logf("decode goldens: %d accepted, %d rejected-as-expected", accepted, rejected)
}

func TestEncodeRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		pix  []byte
		w, h int
	}{
		{"zero width", make([]byte, 0), 0, 4},
		{"zero height", make([]byte, 0), 4, 0},
		{"negative", make([]byte, 64), -4, 4},
		{"short buffer", make([]byte, 4*4*4-1), 4, 4},
		{"long buffer", make([]byte, 4*4*4+1), 4, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EncodeDXT5DDS(c.pix, c.w, c.h); got != nil {
				t.Errorf("expected nil, got %d bytes", len(got))
			}
		})
	}
}

func TestFastDecodeNeverPanics(t *testing.T) {
	seeds := [][]byte{
		nil, {}, []byte("DDS "), []byte("DDS  "),
		bytes.Repeat([]byte{0xff}, 128),
		bytes.Repeat([]byte{0x00}, 128),
	}
	golden, err := os.ReadFile("testdata/dec/ok_bgra.dds")
	if err == nil {
		seeds = append(seeds, golden)

		for n := 0; n <= len(golden); n++ {
			seeds = append(seeds, golden[:n])
		}
		for i := 0; i < len(golden); i++ {
			m := bytes.Clone(golden)
			m[i] ^= 0xff
			seeds = append(seeds, m)
		}
	}
	for i, s := range seeds {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("seed %d (%d bytes) panicked: %v", i, len(s), rec)
				}
			}()
			_, _, _, _ = FastDecodeDDS(s)
		}()
	}
}

func TestEncodeDeterministic(t *testing.T) {
	pix, w, h := readBlob(t, "testdata/enc/rand_s3_67x65.rgba")
	first := EncodeDXT5DDS(pix, w, h)
	for i := 0; i < 5; i++ {
		if !bytes.Equal(EncodeDXT5DDS(pix, w, h), first) {
			t.Fatalf("run %d differs", i)
		}
	}
}

func TestBlockGeometry(t *testing.T) {
	for _, c := range []struct{ w, h, blocks int }{
		{1, 1, 1}, {4, 4, 1}, {5, 4, 2}, {4, 5, 2}, {5, 5, 4},
		{16, 16, 16}, {67, 65, 17 * 17}, {512, 512, 128 * 128},
	} {
		got := EncodeDXT5DDS(make([]byte, c.w*c.h*4), c.w, c.h)
		want := 128 + c.blocks*16
		if len(got) != want {
			t.Errorf("%dx%d: %d bytes, want %d", c.w, c.h, len(got), want)
		}
		if hw := binary.LittleEndian.Uint32(got[16:20]); hw != uint32(c.w) {
			t.Errorf("%dx%d: header width %d", c.w, c.h, hw)
		}
		if hh := binary.LittleEndian.Uint32(got[12:16]); hh != uint32(c.h) {
			t.Errorf("%dx%d: header height %d", c.w, c.h, hh)
		}
		if p := binary.LittleEndian.Uint32(got[20:24]); p != uint32((c.w+3)*4) {
			t.Errorf("%dx%d: header pitch %d, want %d", c.w, c.h, p, (c.w+3)*4)
		}
		if fc := string(got[84:88]); fc != "DXT5" {
			t.Errorf("%dx%d: fourcc %q", c.w, c.h, fc)
		}
	}
}

func BenchmarkEncodeDXT5DDS512(b *testing.B) {
	const w, h = 512, 512
	pix := make([]byte, w*h*4)
	for i := range pix {
		pix[i] = byte(i * 31)
	}
	b.SetBytes(int64(len(pix)))
	for b.Loop() {
		EncodeDXT5DDS(pix, w, h)
	}
}

func BenchmarkFastDecodeDDS512(b *testing.B) {
	const w, h = 512, 512
	data := make([]byte, 128+w*h*4)
	copy(data[0:4], "DDS ")
	binary.LittleEndian.PutUint32(data[4:8], 124)
	binary.LittleEndian.PutUint32(data[8:12], 0x1|0x2|0x4|0x1000)
	binary.LittleEndian.PutUint32(data[12:16], h)
	binary.LittleEndian.PutUint32(data[16:20], w)
	binary.LittleEndian.PutUint32(data[76:80], 32)
	binary.LittleEndian.PutUint32(data[80:84], ddpfRGB|ddpfAlphaPixels)
	binary.LittleEndian.PutUint32(data[88:92], 32)
	binary.LittleEndian.PutUint32(data[92:96], 0x00FF0000)
	binary.LittleEndian.PutUint32(data[96:100], 0x0000FF00)
	binary.LittleEndian.PutUint32(data[100:104], 0x000000FF)
	binary.LittleEndian.PutUint32(data[104:108], 0xFF000000)
	binary.LittleEndian.PutUint32(data[108:112], ddscapsTexture)
	b.SetBytes(int64(w * h * 4))
	for b.Loop() {
		if _, _, _, ok := FastDecodeDDS(data); !ok {
			b.Fatal("not handled")
		}
	}
}
