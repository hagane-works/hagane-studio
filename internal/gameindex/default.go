package gameindex

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	defaultMu    sync.RWMutex
	defaultIndex *Index
)

func SetDefault(x *Index) (restore func()) {
	defaultMu.Lock()
	previous := defaultIndex
	defaultIndex = x
	defaultMu.Unlock()
	return func() {
		defaultMu.Lock()
		defaultIndex = previous
		defaultMu.Unlock()
	}
}

func Default() *Index {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultIndex
}

func Get(ctx context.Context, name string) ([]byte, error) { return Default().Get(ctx, name) }

func Peek(name string) ([]byte, error) { return Default().Peek(name) }

func Load[T any](ctx context.Context, key any, name string, decode func([]byte) (T, error)) (T, error) {
	return Value(ctx, Default(), key, name, decode)
}

func Static(datasets map[string][]byte) *Index {
	r := &run{
		id: "static", state: "ready", static: true, cancel: func() {},
		data: map[string][]byte{}, errs: map[string]error{}, ready: map[string]chan struct{}{},
		values: map[any]*value{}, builtAt: time.Now().UTC().Truncate(time.Second),
		over: make(chan struct{}),
	}
	close(r.over)
	for name, data := range datasets {
		ch := make(chan struct{})
		close(ch)
		r.ready[name] = ch
		r.data[name] = data
		r.done++
		r.total++
	}
	return &Index{static: r, byName: map[string]int{}}
}

func UseStatic(datasets map[string][]byte) (restore func()) {
	return SetDefault(Static(datasets))
}

func CheckReady(name string) error {
	_, err := Peek(name)
	if errors.Is(err, ErrNoGame) {
		return nil
	}
	return err
}
