package exportengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"
)

func tolerancePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xcc
	}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func twoIdeaSnapshot() Snapshot {
	return Snapshot{
		ProjectID: 42,
		Project:   []byte(`{"id":42,"name":"Demo","country_tag":"GER","export_languages":["en"]}`),
		Tables: map[string][]json.RawMessage{
			"ideas": {
				json.RawMessage(`{"id":1,"idea_id":"GER_broken","country_tag":"GER","idea_type":"country","name_en":"Broken","source":"created"}`),
				json.RawMessage(`{"id":2,"idea_id":"GER_good","country_tag":"GER","idea_type":"country","name_en":"Good","source":"created"}`),
			},
			"entity_obs_paths": {
				json.RawMessage(`{"entity_type":"idea_picture","entity_id":1,"obs_key":"idea_broken.png"}`),
				json.RawMessage(`{"entity_type":"idea_picture","entity_id":2,"obs_key":"idea_good.png"}`),
			},
		},
	}
}

const (
	brokenIdeaDDS = "Demo/gfx/interface/ideas/idea_GER_ger_broken.dds"
	goodIdeaDDS   = "Demo/gfx/interface/ideas/idea_GER_ger_good.dds"
	ideaScript    = "Demo/common/ideas/ger_ger.txt"
)

func TestBuildNativeShadowSurvivesOneUnusableImage(t *testing.T) {
	good := tolerancePNG(t)
	cases := []struct {
		name  string
		blob  []byte
		fetch error
	}{
		{name: "zero bytes", blob: []byte{}},
		{name: "nil bytes", blob: nil},
		{name: "not an image", blob: []byte("this is not a picture, it is prose")},
		{name: "truncated png header", blob: good[:8]},
		{name: "fetch failed", fetch: fmt.Errorf("obs: object not found")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			artifact, err := BuildNativeShadowWithBlobs(twoIdeaSnapshot(), nil, false,
				func(key string, _ int64) ([]byte, error) {
					if key == "idea_broken.png" {
						return tc.blob, tc.fetch
					}
					return good, nil
				})

			if err != nil {
				t.Fatalf("one bad image aborted the entire export: %v", err)
			}

			entries := map[string][]byte{}
			for _, entry := range artifact.Entries {
				entries[entry.Path] = entry.Bytes
			}

			if _, present := entries[brokenIdeaDDS]; present {
				t.Fatalf("unusable image was written to the artifact at %s", brokenIdeaDDS)
			}

			script := string(entries[ideaScript])
			if strings.Contains(script, "GER_ger_broken") {
				t.Fatalf("skipped picture still referenced in script:\n%s", script)
			}

			if !bytes.HasPrefix(entries[goodIdeaDDS], []byte("DDS ")) {
				t.Fatalf("one bad image cost the good one too: %s missing", goodIdeaDDS)
			}
			if !strings.Contains(script, "picture = GER_ger_good") {
				t.Fatalf("good idea lost its picture reference:\n%s", script)
			}
			if !strings.Contains(script, "GER_broken") {
				t.Fatalf("idea itself was dropped, not just its picture:\n%s", script)
			}

			var warned bool
			for _, w := range artifact.Warnings {
				if strings.Contains(w, "GER_broken") {
					warned = true
				}
			}
			if !warned {
				t.Fatalf("skipped image was not reported; warnings=%v", artifact.Warnings)
			}
		})
	}
}

func TestBuildNativeShadowFailsClosedWithoutBlobResolver(t *testing.T) {
	_, err := BuildNativeShadow(twoIdeaSnapshot(), nil, false)
	if err == nil {
		t.Fatal("missing blob resolver degraded to a warning; it must abort the export")
	}
	if !strings.Contains(err.Error(), "requires private object resolver") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveImageDDSTolerantClassification(t *testing.T) {
	good := tolerancePNG(t)

	t.Run("resolved", func(t *testing.T) {
		dds, ok, err := ResolveImageDDSTolerant(
			func(string, int64) ([]byte, error) { return good, nil }, "k", "idea demo")
		if err != nil || !ok {
			t.Fatalf("good image did not resolve: ok=%v err=%v", ok, err)
		}
		if !bytes.HasPrefix(dds, []byte("DDS ")) {
			t.Fatalf("not DDS bytes: %q", dds)
		}
	})

	soft := map[string]func(string, int64) ([]byte, error){
		"zero byte object":  func(string, int64) ([]byte, error) { return []byte{}, nil },
		"undecodable bytes": func(string, int64) ([]byte, error) { return []byte("nope"), nil },
		"fetch error":       func(string, int64) ([]byte, error) { return nil, fmt.Errorf("obs down") },
	}
	for name, resolve := range soft {
		t.Run("soft/"+name, func(t *testing.T) {
			dds, ok, err := ResolveImageDDSTolerant(resolve, "k", "idea demo")
			if err != nil {
				t.Fatalf("per-asset failure escalated to a hard error: %v", err)
			}
			if ok || dds != nil {
				t.Fatalf("unusable asset reported as resolved: ok=%v dds=%d bytes", ok, len(dds))
			}
		})
	}

	t.Run("hard/nil resolver", func(t *testing.T) {
		_, ok, err := ResolveImageDDSTolerant(nil, "k", "idea demo")
		if err == nil {
			t.Fatal("nil resolver must be a hard failure")
		}
		if ok {
			t.Fatal("nil resolver reported resolved")
		}
	})
}

func TestResolveImageDDSTolerantKeepsFetchBound(t *testing.T) {
	var seen int64
	_, _, err := ResolveImageDDSTolerant(func(_ string, maxBytes int64) ([]byte, error) {
		seen = maxBytes
		return tolerancePNG(t), nil
	}, "k", "idea demo")
	if err != nil {
		t.Fatal(err)
	}
	if seen != 32<<20 {
		t.Fatalf("fetch bound drifted: got %d, want %d", seen, int64(32<<20))
	}
}
