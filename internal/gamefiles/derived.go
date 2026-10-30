package gamefiles

import "sync"

type derivedValue struct {
	once  sync.Once
	value any
}

func Derived[T any](g *Game, key any, build func(*Game) T) T {
	g.derivedMu.Lock()
	entry, ok := g.derived[key]
	if !ok {
		if g.derived == nil {
			g.derived = map[any]*derivedValue{}
		}
		entry = &derivedValue{}
		g.derived[key] = entry
	}
	g.derivedMu.Unlock()
	entry.once.Do(func() { entry.value = build(g) })
	value, _ := entry.value.(T)
	return value
}
