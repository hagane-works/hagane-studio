package locout

import (
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var backslashDoublingAllowed = map[string]string{
	"emit/misc/units.go": "Clausewitz 脚本 token 的引号包裹(common/units/*.txt)," +
		"不是 loc 值:它按 Python 侧同一规则先转反斜杠再转引号,`/` 只触发加引号不被转义。",
	"emit/moddesc/descriptor.go": "descriptor.mod 的 `name=\"...\"`,Clausewitz 键值文件不是 yml。",
	"exportengine/native.go": "quoteScript():Clausewitz 脚本里的带引号字符串" +
		"(effect/trigger 文本),与 loc 值是两套语法。",
	"importsvc/toscript.go": "EmitQuoted():Clausewitz 脚本字符串的引号包裹" +
		"(导入脚本的往返),不是 loc 值。",
}

func TestBackslashDoublingLivesOnlyInThisPackage(t *testing.T) {

	goRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("locate internal/hoi4: %v", err)
	}
	if _, err := os.Stat(filepath.Join(goRoot, "emit", "locout")); err != nil {
		t.Fatalf("internal/hoi4 root looks wrong (%s): %v", goRoot, err)
	}
	var offenders []string
	scanned := 0
	err = filepath.WalkDir(goRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "testdata", "vendor", "node_modules":
				return fs.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanned++
		if !hasBackslashDoublingPair(body) {
			return nil
		}
		rel := filepath.ToSlash(mustRel(goRoot, path))
		if _, ok := backslashDoublingAllowed[rel]; ok {
			return nil
		}
		offenders = append(offenders, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk go/: %v", err)
	}
	if scanned < 200 {
		t.Fatalf("扫描面塌了:只看了 %d 个 .go 文件,期望 200+(装个漏的闸比不装更坏)", scanned)
	}

	var stale []string
	for rel := range backslashDoublingAllowed {
		full := filepath.Join(goRoot, filepath.FromSlash(rel))
		body, readErr := os.ReadFile(full)
		if readErr != nil {
			stale = append(stale, rel+"(文件已不存在)")
			continue
		}
		if !hasBackslashDoublingPair(body) {
			stale = append(stale, rel+"(已不再翻倍反斜杠)")
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("backslashDoublingAllowed 有过期条目,删掉它们:\n  %s", strings.Join(stale, "\n  "))
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("loc 转义只允许有一份实现(emit/locout/yml.go),但这些文件也在把一个反斜杠翻倍成两个:\n  %s\n"+
			"  这正是 W5 P0-b 修掉的病:作者按 HOI4 惯例写的两字符 `\\n` 被翻倍成 `\\\\n`,游戏里显示字面 \\n。\n"+
			"  发 loc 的改法:转调 locout.YmlEscapeValue(见 emit/misc/loc.go · emit/worldout/yml.go ·\n"+
			"  exportmerge/localisation.go · exportengine/gui_overlay.go)。\n"+
			"  确实不是 loc 值的,在 backslashDoublingAllowed 里加一行并写清理由。",
			strings.Join(offenders, "\n  "))
	}
}

func hasBackslashDoublingPair(src []byte) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("x.go", fset.Base(), len(src))
	var sc scanner.Scanner
	sc.Init(file, src, nil, 0)
	prev := ""
	for {
		_, tok, lit := sc.Scan()
		if tok == token.EOF {
			return false
		}
		if tok != token.STRING {

			if tok != token.COMMA {
				prev = ""
			}
			continue
		}
		value, err := strconv.Unquote(lit)
		if err != nil {
			prev = ""
			continue
		}
		if prev == `\` && value == `\\` {
			return true
		}
		prev = value
	}
}

func mustRel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}
