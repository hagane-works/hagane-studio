package exportengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRawBinaryBytesRemainFailClosedAfterQualityAcknowledgementPolicy(t *testing.T) {
	snapshot := Snapshot{
		ProjectID: 7,
		Project: json.RawMessage(
			`{"id":7,"name":"Demo","country_tag":"GER","export_languages":["en"]}`,
		),
		Tables: map[string][]json.RawMessage{
			"project_raw_files": {
				json.RawMessage(`{
					"id":91,"dest_path":"gfx/interface/missing.dds",
					"source":"created","raw_kind":"verbatim_override",
					"is_binary":true,"content_obs_key":"raw/missing.dds"
				}`),
			},
		},
	}

	_, err := BuildNativeShadowWithBlobs(snapshot, nil, false, nil)
	if err == nil {
		t.Fatal("raw binary content without a byte resolver produced an artifact")
	}
	if !strings.Contains(err.Error(), "requires private object") {
		t.Fatalf("unexpected fail-closed error: %v", err)
	}
}
