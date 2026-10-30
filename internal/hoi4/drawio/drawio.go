package drawio

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	MaxBytes          = 8 * 1024 * 1024
	pxPerGridUnit     = 100.0
	maxFocusIDRunes   = 200
	maxFocusNameRunes = 255
)

var (
	tagRE = regexp.MustCompile(`<[^>]*>`)
)

type Node struct {
	FocusID string `json:"focus_id"`
	Name    string `json:"name"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Result struct {
	Nodes    []Node   `json:"nodes"`
	Edges    []Edge   `json:"edges"`
	Warnings []string `json:"warnings"`
}

type element struct {
	name     xml.Name
	attrs    []xml.Attr
	children []*element
}

func (e *element) tagIs(tag string) bool {

	return e != nil && e.name.Space == "" && e.name.Local == tag
}

func (e *element) attr(name string) string {
	value, _ := e.attrOK(name)
	return value
}

func (e *element) attrOK(name string) (string, bool) {
	if e == nil {
		return "", false
	}
	for _, attr := range e.attrs {
		if attr.Name.Space == "" && attr.Name.Local == name {
			return attr.Value, true
		}
	}
	return "", false
}

func (e *element) firstChild(tag string) *element {
	if e == nil {
		return nil
	}
	for _, child := range e.children {
		if child.tagIs(tag) {
			return child
		}
	}
	return nil
}

func descendants(root *element, tag string) []*element {
	var out []*element
	var walk func(*element)
	walk = func(node *element) {
		for _, child := range node.children {
			if child.tagIs(tag) {
				out = append(out, child)
			}
			walk(child)
		}
	}
	if root != nil {
		walk(root)
	}
	return out
}

func parseXML(text string) (*element, error) {

	decoder := xml.NewDecoder(strings.NewReader(text))
	decoder.Strict = true
	var root *element
	stack := make([]*element, 0, 16)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Draw.io XML parse failed: %w", err)
		}
		switch value := token.(type) {
		case xml.Directive:
			upper := strings.ToUpper(strings.TrimSpace(string(value)))
			if strings.HasPrefix(upper, "DOCTYPE") || strings.HasPrefix(upper, "ENTITY") {
				return nil, fmt.Errorf("Draw.io XML rejected (unsafe XML construct): DOCTYPE and ENTITY declarations are not allowed")
			}
		case xml.StartElement:
			node := &element{name: value.Name, attrs: append([]xml.Attr(nil), value.Attr...)}
			if len(stack) == 0 {
				if root != nil {
					return nil, fmt.Errorf("Draw.io XML parse failed: multiple root elements")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("Draw.io XML parse failed: unexpected closing element")
			}
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil {
		return nil, fmt.Errorf("Draw.io XML parse failed: no root element")
	}
	return root, nil
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes)
}

func plainLabel(label string) string {
	withoutTags := tagRE.ReplaceAllString(label, " ")

	cleaned := strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
	return strings.TrimRightFunc(truncateRunes(cleaned, maxFocusNameRunes), unicode.IsSpace)
}

func sanitizeFocusID(label string) string {
	var out strings.Builder
	lastUnsafe := false
	for _, r := range label {
		safe := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if safe {
			out.WriteRune(r)
			lastUnsafe = false
		} else if !lastUnsafe {
			out.WriteByte('_')
			lastUnsafe = true
		}
	}
	cleaned := strings.Trim(out.String(), "_")
	cleaned = strings.Trim(truncateRunes(cleaned, maxFocusIDRunes), "_")
	if cleaned == "" {
		return "focus"
	}
	return cleaned
}

type cellView struct {
	id, label, vertex, edge, source, target string
	geometry                                *element
}

func viewCell(cell *element) cellView {
	if cell.tagIs("object") || cell.tagIs("UserObject") {
		inner := cell.firstChild("mxCell")
		label := cell.attr("label")
		if label == "" {
			label = cell.attr("value")
		}
		view := cellView{id: cell.attr("id"), label: strings.TrimSpace(label)}
		if inner != nil {
			view.vertex = inner.attr("vertex")
			view.edge = inner.attr("edge")
			view.geometry = inner.firstChild("mxGeometry")
		}
		view.source = cell.attr("source")
		if view.source == "" && inner != nil {
			view.source = inner.attr("source")
		}
		view.target = cell.attr("target")
		if view.target == "" && inner != nil {
			view.target = inner.attr("target")
		}
		return view
	}
	return cellView{
		id: cell.attr("id"), label: strings.TrimSpace(cell.attr("value")),
		vertex: cell.attr("vertex"), edge: cell.attr("edge"),
		source: cell.attr("source"), target: cell.attr("target"),
		geometry: cell.firstChild("mxGeometry"),
	}
}

func geometry(view cellView) (float64, float64, bool) {
	if view.geometry == nil {
		return 0, 0, true
	}
	parse := func(value string) (float64, error) {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, fmt.Errorf("invalid number")
		}
		return parsed, nil
	}
	xRaw, hasX := view.geometry.attrOK("x")
	if !hasX {
		xRaw = "0"
	}
	yRaw, hasY := view.geometry.attrOK("y")
	if !hasY {
		yRaw = "0"
	}
	x, xerr := parse(xRaw)
	y, yerr := parse(yRaw)
	if xerr != nil || yerr != nil {
		return 0, 0, false
	}
	return x, y, true
}

func Parse(text, prefix string) (Result, error) {
	result := Result{Nodes: []Node{}, Edges: []Edge{}, Warnings: []string{}}
	if strings.TrimSpace(text) == "" {
		return result, fmt.Errorf("empty Draw.io payload")
	}
	byteLen := len([]byte(text))
	if byteLen > MaxBytes {
		return result, fmt.Errorf("Draw.io payload too large: %d bytes exceeds cap of %d bytes", byteLen, MaxBytes)
	}
	root, err := parseXML(text)
	if err != nil {
		return result, err
	}

	wrappers := append(descendants(root, "object"), descendants(root, "UserObject")...)
	wrappedInner := map[*element]bool{}
	for _, wrapper := range wrappers {
		for _, child := range wrapper.children {
			if child.tagIs("mxCell") {
				wrappedInner[child] = true
			}
		}
	}
	cells := append([]*element{}, wrappers...)
	for _, cell := range descendants(root, "mxCell") {
		if !wrappedInner[cell] {
			cells = append(cells, cell)
		}
	}

	idToFocusID := map[string]string{}
	usedIDs := map[string]bool{}
	claimID := func(base string) string {
		candidate := base
		for suffix := 2; usedIDs[candidate]; suffix++ {
			candidate = fmt.Sprintf("%s_%d", base, suffix)
		}
		usedIDs[candidate] = true
		return candidate
	}

	for _, cell := range cells {
		view := viewCell(cell)
		if view.vertex != "1" {
			continue
		}
		rawX, rawY, validGeometry := geometry(view)
		if !validGeometry {
			rawX, rawY = 0, 0
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("cell %s: invalid geometry · using (0,0)", view.id))
		}
		label := plainLabel(view.label)
		focusID := ""
		if label == "" {
			cellID := view.id
			if cellID == "" {
				cellID = "unnamed"
			}
			focusID = "focus_" + cellID
		} else {
			focusID = sanitizeFocusID(label)
		}
		if prefix != "" {
			focusID = prefix + "_" + focusID
		}
		claimed := claimID(focusID)
		if claimed != focusID {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("focus_id collision · '%s' renamed to '%s'", focusID, claimed))
		}
		idToFocusID[view.id] = claimed
		name := label
		if name == "" {
			name = claimed
		}
		result.Nodes = append(result.Nodes, Node{
			FocusID: claimed, Name: name,
			X: int(math.Floor(rawX / pxPerGridUnit)),
			Y: int(math.Floor(rawY / pxPerGridUnit)),
		})
	}

	if len(result.Nodes) > 0 {
		minX, minY := result.Nodes[0].X, result.Nodes[0].Y
		for _, node := range result.Nodes[1:] {
			if node.X < minX {
				minX = node.X
			}
			if node.Y < minY {
				minY = node.Y
			}
		}
		if minX < 0 || minY < 0 {
			for i := range result.Nodes {
				result.Nodes[i].X -= minX
				result.Nodes[i].Y -= minY
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("negative canvas coords · shifted all nodes by (%d,%d) to keep grid positions non-negative", -minX, -minY))
		}
	}

	occupied := map[[2]int]bool{}
	for i := range result.Nodes {
		node := &result.Nodes[i]
		position := [2]int{node.X, node.Y}
		if !occupied[position] {
			occupied[position] = true
			continue
		}
		originalX, originalY := node.X, node.Y
		nx, ny := originalX, originalY
		for occupied[[2]int{nx, ny}] {
			nx++
			if nx-originalX > 64 {
				nx = originalX
				ny++
			}
		}
		occupied[[2]int{nx, ny}] = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("grid collision · focus '%s' moved from (%d,%d) to (%d,%d) to avoid overlap",
				node.FocusID, originalX, originalY, nx, ny))
		node.X, node.Y = nx, ny
	}

	for _, cell := range cells {
		view := viewCell(cell)
		if view.edge != "1" {
			continue
		}
		source, sourceOK := idToFocusID[view.source]
		target, targetOK := idToFocusID[view.target]
		if !sourceOK || !targetOK {
			cellID := view.id
			if cellID == "" {
				cellID = "?"
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("edge %s: dangling source / target · skipped", cellID))
			continue
		}
		if source == target {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("self-loop edge on '%s' · dropped (a focus cannot be its own prerequisite)", source))
			continue
		}
		result.Edges = append(result.Edges, Edge{Source: source, Target: target})
	}
	return result, nil
}

func SanitizeFocusID(value string) string { return sanitizeFocusID(value) }
