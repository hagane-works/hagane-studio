package worldout

import (
	"reflect"
	"strings"
	"testing"
)

func TestEnsureEnFirst_MatchesPythonReference(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{

			name: "空列表 = 作者要 EN-only,不许自作主张加语言",
			in:   nil,
			want: []string{"en"},
		},
		{
			name: "空切片同上",
			in:   []string{},
			want: []string{"en"},
		},
		{
			name: "en 已在首位则原样",
			in:   []string{"en", "zh"},
			want: []string{"en", "zh"},
		},
		{
			name: "en 在别处 → 提到首位",
			in:   []string{"zh", "en", "ru"},
			want: []string{"en", "zh", "ru"},
		},
		{
			name: "en 缺席 → 强制补到首位(HOI4 靠 english 兜底)",
			in:   []string{"zh", "ru"},
			want: []string{"en", "zh", "ru"},
		},
		{
			name: "重复折叠,保留首次出现",
			in:   []string{"zh", "zh", "ru", "zh"},
			want: []string{"en", "zh", "ru"},
		},
		{
			name: "未知语言码静默丢弃(API 已在 422 拦过,这里是纵深防御)",
			in:   []string{"zh", "klingon", "ru"},
			want: []string{"en", "zh", "ru"},
		},
		{
			name: "空串/空白项跳过",
			in:   []string{"", "  ", "zh"},
			want: []string{"en", "zh"},
		},
		{
			name: "大小写与首尾空白归一",
			in:   []string{" ZH ", "Ru"},
			want: []string{"en", "zh", "ru"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ensureEnFirst(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ensureEnFirst(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestEmptyLanguagesEmitsNoNonEnglishLocFile(t *testing.T) {
	triples := buildExportLangTriples(nil)
	if len(triples) != 1 {
		t.Fatalf("空语言列表应只产出 1 组三元组,得到 %d 组: %+v", len(triples), triples)
	}
	got := triples[0]
	if got.Code != "en" {
		t.Errorf("Code = %q, want \"en\"", got.Code)
	}

	if got.Subdir != "english/replace" {
		t.Errorf("Subdir = %q, want \"english/replace\"", got.Subdir)
	}
	if got.Suffix != "l_english" {
		t.Errorf("Suffix = %q, want \"l_english\"", got.Suffix)
	}
	for _, tr := range triples {
		if strings.Contains(tr.Subdir, "simp_chinese") {
			t.Fatalf("作者只要英文,却仍会写 %s", tr.Subdir)
		}
	}
}
