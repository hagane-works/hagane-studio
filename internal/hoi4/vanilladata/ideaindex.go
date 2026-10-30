package vanilladata

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"haganeworks/internal/gameindex"
)

type ideaPictureSet struct {
	set    map[string]struct{}
	sorted []string

	lower []string
}

type ideaPicturesKey struct{}

var emptyIdeaPictures = &ideaPictureSet{set: map[string]struct{}{}}

func ideaPictureNamesOnce() *ideaPictureSet {
	set, err := gameindex.Load(context.Background(), ideaPicturesKey{}, gameindex.IdeaIndex, decodeIdeaPictures)
	if err != nil {
		return emptyIdeaPictures
	}
	return set
}

func decodeIdeaPictures(raw []byte) (*ideaPictureSet, error) {
	var top struct {
		PictureSprites []string `json:"picture_sprites"`
		Ideas          []struct {
			Picture string `json:"picture"`
		} `json:"ideas"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}

	names := make(map[string]struct{}, len(top.PictureSprites)+len(top.Ideas))
	add := func(raw string) {
		n := strings.TrimSpace(raw)
		n = strings.TrimPrefix(n, "GFX_idea_")
		if n != "" {
			names[n] = struct{}{}
		}
	}
	for _, sprite := range top.PictureSprites {
		add(sprite)
	}
	for _, idea := range top.Ideas {
		add(idea.Picture)
	}

	out := &ideaPictureSet{set: names, sorted: make([]string, 0, len(names))}
	for n := range names {
		out.sorted = append(out.sorted, n)
	}
	sort.Strings(out.sorted)
	out.lower = make([]string, len(out.sorted))
	for i, n := range out.sorted {
		out.lower[i] = strings.ToLower(n)
	}
	return out, nil
}

func IsVanillaIdeaPicture(name string) bool {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "GFX_idea_")
	if name == "" {
		return false
	}
	_, ok := ideaPictureNamesOnce().set[name]
	return ok
}

func VanillaIdeaPictureCount() int { return len(ideaPictureNamesOnce().set) }

const suggestScoreFloor = 55.0

func SuggestIdeaPictures(name string, n int) []string {
	if n <= 0 {
		return nil
	}
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "GFX_idea_")
	if name == "" {
		return nil
	}
	q := strings.ToLower(name)

	idx := ideaPictureNamesOnce()
	type cand struct {
		name  string
		score float64
	}
	scored := make([]cand, 0, 64)
	for i, lc := range idx.lower {
		s := suggestScore(q, lc)
		if s >= suggestScoreFloor {
			scored = append(scored, cand{name: idx.sorted[i], score: s})
		}
	}
	if len(scored) == 0 {
		return nil
	}
	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].score != scored[b].score {
			return scored[a].score > scored[b].score
		}
		return scored[a].name < scored[b].name
	})
	if len(scored) > n {
		scored = scored[:n]
	}
	out := make([]string, len(scored))
	for i, c := range scored {
		out[i] = c.name
	}
	return out
}

func suggestScore(query, candidate string) float64 {
	if query == candidate {
		return 125
	}
	maxLen := len(query)
	if len(candidate) > maxLen {
		maxLen = len(candidate)
	}
	if maxLen == 0 {
		return 0
	}

	sim := 0.0
	if d := levenshtein(query, candidate); d < maxLen {
		sim = (1 - float64(d)/float64(maxLen)) * 100
	}
	switch {
	case strings.HasPrefix(candidate, query) || strings.HasPrefix(query, candidate):
		sim += 25
	case strings.Contains(candidate, query) || strings.Contains(query, candidate):
		sim += 15
	}
	if sim > 124 {
		sim = 124
	}
	return sim
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
