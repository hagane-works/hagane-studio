package exportengine

import (
	"reflect"
	"testing"
)

func TestCollectLocalisationBucketsFirstSeenAndLocaleIsolation(t *testing.T) {
	entries := []Entry{
		{Path: "Demo/localisation/english/a.yml", Bytes: []byte("\ufeffl_english:\n KEY:0 \"first\"\n")},
		{Path: "Demo/localisation/english/b.yml", Bytes: []byte("l_english:\n KEY:0 \"second\"\n OTHER:0 \"value\"\n")},
		{Path: "Demo/localisation/simp_chinese/a.yml", Bytes: []byte("l_simp_chinese:\n KEY:0 \"中文\"\n")},
		{Path: "Demo/common/not_loc.yml", Bytes: []byte("l_english:\n LEAK:0 \"no\"\n")},
	}
	buckets := map[string]map[string]string{"en": {}, "zh": {}}
	collectLocalisationBuckets(entries, map[string]bool{"en": true, "zh": true}, buckets)
	if want := map[string]map[string]string{
		"en": {"KEY": "first", "OTHER": "value"},
		"zh": {"KEY": "中文"},
	}; !reflect.DeepEqual(buckets, want) {
		t.Fatalf("buckets=%v want=%v", buckets, want)
	}
}
