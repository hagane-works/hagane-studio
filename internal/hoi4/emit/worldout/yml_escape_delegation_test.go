package worldout

import (
	"testing"

	"haganeworks/internal/hoi4/emit/locout"
)

var worldEscapeAlphabet = []string{`\`, "n", "t", `"`, "a", "\n", "\t"}

func TestWorldoutEscapeDelegatesToLocout(t *testing.T) {
	checked := 0
	reported := false
	var walk func(prefix string, depth int)
	walk = func(prefix string, depth int) {
		if prefix != "" {
			checked++
			if got, want := ymlEscapeValue(prefix), locout.YmlEscapeValue(prefix); got != want && !reported {
				reported = true
				t.Errorf("worldout 又长出了自己的转义实现:input=%q worldout=%q locout=%q\n"+
					"  州名/州描述/cosmetic tag 国名的 loc 全部走它,两份不一致 = 产物不一致。",
					prefix, got, want)
			}
		}
		if depth == 0 {
			return
		}
		for _, c := range worldEscapeAlphabet {
			walk(prefix+c, depth-1)
		}
	}
	walk("", 4)
	if checked < 2800 {
		t.Fatalf("穷举面塌了:只检查了 %d 条,期望 2800", checked)
	}
}

func TestWorldoutAuthoredBackslashN(t *testing.T) {
	const in = `First line.\nSecond line.`
	if got := ymlEscapeValue(in); got != in {
		t.Errorf("州/国名描述写成 %q 被发成 %q,游戏里显示字面 \\n", in, got)
	}
	if got := ymlEscapeValue(`C:\mods\`); got != `C:\\mods\\` {
		t.Errorf("H-35 回归:%q 被发成 %q,期望 %q", `C:\mods\`, got, `C:\\mods\\`)
	}
}
