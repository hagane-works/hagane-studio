package importsvc

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"strings"
	"testing"

	"haganeworks/internal/hoi4/paradox"
)

func TestFloatScriptScalarMatchesPythonWide(t *testing.T) {
	const path = "testdata/float_script_scalar_python.tsv"
	f, err := os.Open(path)
	if err != nil {

		t.Fatalf("missing reference fixture %s: %v", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n, bad := 0, 0
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed fixture line %q", line)
		}
		raw, err := hex.DecodeString(parts[0])
		if err != nil || len(raw) != 8 {
			t.Fatalf("bad float64 hex %q", parts[0])
		}
		v := math.Float64frombits(binary.BigEndian.Uint64(raw))
		got := fmtScriptScalar(paradox.Scalar{Kind: paradox.ScalarFloat, Float: v}, false)
		if got != parts[1] {
			bad++
			if bad <= 10 {
				t.Errorf("bits=%s: Go=%q Python=%q", parts[0], got, parts[1])
			}
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if n < 600 {
		t.Fatalf("fixture only had %d rows — expected ~617; was it truncated?", n)
	}
	if bad != 0 {
		t.Errorf("%d/%d values disagree with CPython", bad, n)
	}
}
