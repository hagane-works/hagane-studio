package focusorder

import (
	"container/heap"
	"sort"
)

type Node struct {
	FocusID            string
	RelativePositionID string
}

type Diagnostic struct {
	Code               string `json:"code"`
	NodeIndices        []int  `json:"node_indices"`
	AffectedIndices    []int  `json:"affected_indices"`
	FocusID            string `json:"focus_id,omitempty"`
	RelativePositionID string `json:"relative_position_id,omitempty"`
}

type Result struct {
	Order       []int
	Diagnostics []Diagnostic
}

func Plan(nodes []Node) Result {
	result := Result{Order: make([]int, 0, len(nodes)), Diagnostics: []Diagnostic{}}
	byID := make(map[string][]int, len(nodes))
	components := newComponents(len(nodes))
	anchors := make([]int, len(nodes))
	for i, node := range nodes {
		anchors[i] = -1
		if node.FocusID == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Code: "focus_layout_missing_focus_id", NodeIndices: []int{i},
			})
			continue
		}
		byID[node.FocusID] = append(byID[node.FocusID], i)
	}

	for i, node := range nodes {
		indices := byID[node.FocusID]
		if len(indices) < 2 || indices[0] != i {
			continue
		}
		for _, duplicate := range indices[1:] {
			components.join(i, duplicate)
		}
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "focus_layout_duplicate_focus_id", FocusID: node.FocusID,
			NodeIndices: append([]int(nil), indices...),
		})
	}
	for i, node := range nodes {
		if node.RelativePositionID == "" {
			continue
		}
		targets := byID[node.RelativePositionID]
		if len(targets) == 0 {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Code: "focus_layout_missing_anchor", NodeIndices: []int{i},
				FocusID: node.FocusID, RelativePositionID: node.RelativePositionID,
			})
			continue
		}
		components.join(i, targets[0])
		if len(targets) == 1 {
			anchors[i] = targets[0]
		}
	}

	state := make([]uint8, len(nodes))
	pathPosition := make([]int, len(nodes))
	for start := range nodes {
		if state[start] != 0 {
			continue
		}
		path := []int{}
		current := start
		for current >= 0 && state[current] == 0 {
			state[current] = 1
			pathPosition[current] = len(path)
			path = append(path, current)
			current = anchors[current]
		}
		if current >= 0 && state[current] == 1 {
			cycle := append([]int(nil), path[pathPosition[current]:]...)
			sort.Ints(cycle)
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Code: "focus_layout_relative_cycle", NodeIndices: cycle,
			})
		}
		for _, i := range path {
			state[i] = 2
		}
	}
	groups := make(map[int][]int, len(nodes))
	for i := range nodes {
		root := components.find(i)
		groups[root] = append(groups[root], i)
	}
	invalid := make(map[int]bool)
	for i := range result.Diagnostics {
		diagnostic := &result.Diagnostics[i]
		root := components.find(diagnostic.NodeIndices[0])
		invalid[root] = true

		diagnostic.AffectedIndices = groups[root]
	}
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		left, right := result.Diagnostics[i], result.Diagnostics[j]
		if left.NodeIndices[0] != right.NodeIndices[0] {
			return left.NodeIndices[0] < right.NodeIndices[0]
		}
		return left.Code < right.Code
	})
	children := make([][]int, len(nodes))
	indegree := make([]int, len(nodes))
	previousInvalid := make(map[int]int)
	for i := range nodes {
		root := components.find(i)
		parent := anchors[i]
		if invalid[root] {
			parent = -1
			if previous, ok := previousInvalid[root]; ok {
				parent = previous
			}
			previousInvalid[root] = i
		}
		if parent >= 0 {
			children[parent] = append(children[parent], i)
			indegree[i]++
		}
	}
	ready := &indexHeap{}
	for i, degree := range indegree {
		if degree == 0 {
			heap.Push(ready, i)
		}
	}
	for ready.Len() > 0 {
		i := heap.Pop(ready).(int)
		result.Order = append(result.Order, i)
		for _, child := range children[i] {
			indegree[child]--
			if indegree[child] == 0 {
				heap.Push(ready, child)
			}
		}
	}
	return result
}

type disjointComponents struct {
	parent []int
	size   []int
}

func newComponents(count int) disjointComponents {
	c := disjointComponents{parent: make([]int, count), size: make([]int, count)}
	for i := range c.parent {
		c.parent[i], c.size[i] = i, 1
	}
	return c
}

func (c disjointComponents) find(i int) int {
	for c.parent[i] != i {
		c.parent[i] = c.parent[c.parent[i]]
		i = c.parent[i]
	}
	return i
}

func (c disjointComponents) join(a, b int) {
	a, b = c.find(a), c.find(b)
	if a == b {
		return
	}
	if c.size[a] < c.size[b] {
		a, b = b, a
	}
	c.parent[b] = a
	c.size[a] += c.size[b]
}

type indexHeap []int

func (h indexHeap) Len() int           { return len(h) }
func (h indexHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h indexHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *indexHeap) Push(value any)    { *h = append(*h, value.(int)) }
func (h *indexHeap) Pop() any {
	old := *h
	value := old[len(old)-1]
	*h = old[:len(old)-1]
	return value
}
