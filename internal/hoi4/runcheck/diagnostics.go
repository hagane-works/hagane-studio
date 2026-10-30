package runcheck

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Event struct {
	ID          int64
	NamespaceID int64
	EventID     string
	EventNumber int64
	Namespace   string
}

type FocusNode struct {
	ID                int64
	TreeID            int64
	TreeKind          string
	FocusID           string
	RelativePosition  string
	Prerequisites     [][]string
	MutuallyExclusive []string
}

type Snapshot struct {
	Events     []Event
	FocusNodes []FocusNode
}

type Finding struct {
	Code        string         `json:"code"`
	EntityRef   map[string]any `json:"entity_ref"`
	Severity    string         `json:"severity"`
	Message     string         `json:"message"`
	FixHint     string         `json:"fix_hint"`
	Detail      map[string]any `json:"detail"`
	Source      map[string]any `json:"source"`
	RepairRoute string         `json:"repair_route"`
}

func finding(code string, ref, detail map[string]any, route string) Finding {
	return Finding{
		Code: code, EntityRef: ref, Severity: "warning", Message: "", FixHint: "",
		Detail: detail, Source: map[string]any{"kind": "go_run_check"}, RepairRoute: route,
	}
}

func Collect(snapshot Snapshot) []Finding {
	findings := collectEvents(snapshot.Events)
	findings = append(findings, collectFocusReferences(snapshot.FocusNodes)...)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return entityID(findings[i]) < entityID(findings[j])
	})
	return findings
}

func collectEvents(events []Event) []Finding {
	var out []Finding
	for _, event := range events {
		id := strings.TrimSpace(event.EventID)
		if id == "" {
			continue
		}
		dot := strings.LastIndex(id, ".")
		idNamespace := ""
		suffix := ""
		if dot > 0 && dot < len(id)-1 {
			idNamespace, suffix = id[:dot], id[dot+1:]
		}
		actual := strings.TrimSpace(event.Namespace)
		ref := map[string]any{
			"type": "event", "id": event.ID, "namespace_id": event.NamespaceID,
		}
		if idNamespace == "" || idNamespace != actual {
			out = append(out, finding(
				"event_namespace_mismatch",
				ref,
				map[string]any{
					"event_id": id, "actual_namespace": actual,
					"id_namespace": emptyLabel(idNamespace),
				},
				"events.identity",
			))
			continue
		}
		number, err := strconv.ParseInt(suffix, 10, 64)
		if err != nil || number != event.EventNumber {
			idNumber := any(suffix)
			if err == nil {
				idNumber = number
			}
			out = append(out, finding(
				"event_number_mismatch",
				ref,
				map[string]any{"event_id": id, "event_number": event.EventNumber, "id_number": idNumber},
				"events.identity",
			))
		}
	}
	return out
}

func collectFocusReferences(nodes []FocusNode) []Finding {
	known := make(map[string]FocusNode, len(nodes))
	byFocusID := make(map[string][]FocusNode, len(nodes))
	for _, node := range nodes {
		if id := strings.TrimSpace(node.FocusID); id != "" {
			known[id] = node
			byFocusID[id] = append(byFocusID[id], node)
		}
	}
	var out []Finding
	for focusID, duplicates := range byFocusID {
		if len(duplicates) < 2 {
			continue
		}
		nodeIDs := make([]int64, 0, len(duplicates))
		for _, node := range duplicates {
			nodeIDs = append(nodeIDs, node.ID)
		}
		sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })
		first := duplicates[0]
		out = append(out, finding(
			"focus_id_duplicate",
			map[string]any{"type": "focus", "id": first.ID, "tree_id": first.TreeID},
			map[string]any{"focus_id": focusID, "count": len(duplicates), "node_ids": nodeIDs},
			"focus.identity",
		))
	}
	for _, node := range nodes {
		focusID := strings.TrimSpace(node.FocusID)
		if focusID == "" {
			continue
		}
		ref := map[string]any{"type": "focus", "id": node.ID, "tree_id": node.TreeID}
		seen := map[string]bool{}
		addMissing := func(code, relationship, target string) {
			target = strings.TrimSpace(target)
			key := code + "\x00" + target
			if target == "" || seen[key] {
				return
			}
			seen[key] = true
			if _, ok := known[target]; ok {
				return
			}
			detail := map[string]any{
				"focus_id": focusID, "reference": target, "relationship": relationship,
			}
			if code == "focus_prereq_not_found" {
				detail["prereq"] = target
			}
			if code == "focus_mutex_not_found" {
				detail["mutex"] = target
			}
			out = append(out, finding(code, ref, detail, "focus.references"))
		}
		for _, group := range node.Prerequisites {
			for _, target := range group {
				addMissing("focus_prereq_not_found", "prerequisite", target)
			}
		}
		for _, target := range node.MutuallyExclusive {
			addMissing("focus_mutex_not_found", "mutually_exclusive", target)
		}
		addMissing("focus_relative_position_not_found", "relative_position", node.RelativePosition)
	}
	out = append(out, collectFocusCycles(nodes, known)...)
	return out
}

func collectFocusCycles(nodes []FocusNode, known map[string]FocusNode) []Finding {
	graph := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		id := strings.TrimSpace(node.FocusID)
		if id == "" {
			continue
		}
		for _, group := range node.Prerequisites {
			for _, target := range group {
				if _, ok := known[target]; ok {
					graph[id] = append(graph[id], target)
				}
			}
		}
	}

	index := 0
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var components [][]string
	var visit func(string)
	visit = func(v string) {
		indices[v], low[v] = index, index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, next := range graph[v] {
			if _, seen := indices[next]; !seen {
				visit(next)
				if low[next] < low[v] {
					low[v] = low[next]
				}
			} else if onStack[next] && indices[next] < low[v] {
				low[v] = indices[next]
			}
		}
		if low[v] != indices[v] {
			return
		}
		var component []string
		for len(stack) > 0 {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == v {
				break
			}
		}
		if len(component) > 1 || hasEdge(graph[v], v) {
			components = append(components, component)
		}
	}

	ids := make([]string, 0, len(known))
	for id := range known {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, seen := indices[id]; !seen {
			visit(id)
		}
	}
	var out []Finding
	for _, component := range components {
		sort.Strings(component)
		node := known[component[0]]
		out = append(out, finding(
			"focus_prereq_cycle",
			map[string]any{"type": "focus", "id": node.ID, "tree_id": node.TreeID},
			map[string]any{"focus_id": node.FocusID, "focus_ids": component},
			"focus.references",
		))
	}
	return out
}

func hasEdge(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func entityID(finding Finding) string { return fmt.Sprint(finding.EntityRef["id"]) }

func emptyLabel(value string) string {
	if value == "" {
		return "(missing)"
	}
	return value
}
