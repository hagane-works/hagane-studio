package importjob

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCosmeticManagerSeam(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "manager.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"stageCosmeticTags": false, "mergeCosmeticTags": false}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if _, exists := want[selector.Sel.Name]; exists {
				want[selector.Sel.Name] = true
			}
		}
		return true
	})
	for name, called := range want {
		if !called {
			t.Errorf("central import seam missing call to %s", name)
		}
	}
}
