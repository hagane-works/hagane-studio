package emitmisc

import (
	"testing"

	"haganeworks/internal/hoi4/emit/locout"
)

var escapeAlphabet = []string{`\`, "n", "t", `"`, "a", "\n", "\t"}

func TestEmitMiscEscapeDelegatesToLocout(t *testing.T) {
	var walk func(prefix string, depth int)
	checked := 0
	var firstDiff string
	walk = func(prefix string, depth int) {
		if prefix != "" {
			checked++
			if got, want := YmlEscapeValue(prefix), locout.YmlEscapeValue(prefix); got != want && firstDiff == "" {
				firstDiff = prefix
				t.Errorf("emit/misc 又长出了自己的转义实现:input=%q emitmisc=%q locout=%q\n"+
					"  国策/科技/决议/装备/兵种/超级事件的 loc 全部走 emit/misc 这一份"+
					"(exportengine/native.go:1485,1737),两份不一致 = 产物不一致。",
					prefix, got, want)
			}
		}
		if depth == 0 {
			return
		}
		for _, c := range escapeAlphabet {
			walk(prefix+c, depth-1)
		}
	}
	walk("", 4)
	if checked < 2800 {
		t.Fatalf("穷举面塌了:只检查了 %d 条,期望 2800(7+49+343+2401)", checked)
	}
}

func TestEmitMiscAuthoredBackslashN(t *testing.T) {
	const in = `First line.\nSecond line.`
	if got := YmlEscapeValue(in); got != in {
		t.Errorf("国策/决议/超级事件描述写成 %q 被发成 %q,游戏里显示字面 \\n 而不是换行", in, got)
	}

	if got := YmlEscapeValue(`C:\mods\`); got != `C:\\mods\\` {
		t.Errorf("H-35 回归:%q 被发成 %q,期望 %q", `C:\mods\`, got, `C:\\mods\\`)
	}
}
