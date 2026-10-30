package gameindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const FormatVersion = 1

const buildWorkers = 4

var ErrNoGame = errors.New("the Hearts of Iron IV folder is not set")

var ErrBuilding = errors.New("the game index is still being built")

var ErrFailed = errors.New("the game index could not be built")

type Dataset struct {
	Name string

	Version int

	Deps []string

	Build func(ctx context.Context, in *Input) ([]byte, error)
}

type Input struct {
	FS fs.FS

	GameVersion string

	deps map[string][]byte
}

func (in *Input) Dataset(name string) []byte { return in.deps[name] }

type Status struct {
	State       string `json:"state"`
	GameVersion string `json:"game_version"`

	Step    string `json:"step"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Error   string `json:"error"`
	BuiltAt string `json:"built_at"`
}

type Index struct {
	source   func() (fs.FS, error)
	datasets []Dataset
	byName   map[string]int

	root string

	mu  sync.Mutex
	run *run

	static *run
}

func New(source func() (fs.FS, error), datasets []Dataset, root string) (*Index, error) {
	ordered, err := order(datasets)
	if err != nil {
		return nil, err
	}
	x := &Index{source: source, datasets: ordered, byName: map[string]int{}, root: root}
	for i, d := range ordered {
		x.byName[d.Name] = i
	}
	return x, nil
}

func DefaultRoot() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Hagane Works", "game-index"), nil
}

func order(datasets []Dataset) ([]Dataset, error) {
	byName := map[string]Dataset{}
	for _, d := range datasets {
		if d.Name == "" || d.Build == nil {
			return nil, fmt.Errorf("gameindex: dataset %q has no name or builder", d.Name)
		}
		if _, dup := byName[d.Name]; dup {
			return nil, fmt.Errorf("gameindex: dataset %q registered twice", d.Name)
		}
		byName[d.Name] = d
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Dataset
	state := map[string]int{}
	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("gameindex: dependency cycle at %q", name)
		case 2:
			return nil
		}
		d, ok := byName[name]
		if !ok {
			return fmt.Errorf("gameindex: unknown dependency %q", name)
		}
		state[name] = 1
		for _, dep := range d.Deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[name] = 2
		out = append(out, d)
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type run struct {
	id          string
	dir         string
	fsys        fs.FS
	gameVersion string
	cancel      context.CancelFunc
	static      bool

	mu      sync.Mutex
	state   string
	step    string
	done    int
	total   int
	err     error
	builtAt time.Time
	data    map[string][]byte
	errs    map[string]error
	ready   map[string]chan struct{}
	values  map[any]*value
	over    chan struct{}
}

type value struct {
	once sync.Once
	v    any
	err  error
}

func (x *Index) Ensure() { x.current() }

func (x *Index) Rebuild() Status {
	if x == nil {
		return Status{State: "missing"}
	}
	if x.static != nil {
		return x.static.status()
	}
	fsys, err := x.open()
	if err != nil {
		x.mu.Lock()
		x.stopLocked()
		x.mu.Unlock()
		return Status{State: "missing"}
	}
	x.mu.Lock()
	x.stopLocked()
	r := x.startLocked(fsys, true)
	x.mu.Unlock()
	return r.status()
}

func (x *Index) Status() Status {
	r := x.current()
	if r == nil {
		return Status{State: "missing"}
	}
	return r.status()
}

func (x *Index) Wait(ctx context.Context) (Status, error) {
	r := x.current()
	if r == nil {
		return Status{State: "missing"}, ErrNoGame
	}
	select {
	case <-r.over:
		return r.status(), nil
	case <-ctx.Done():
		return r.status(), ctx.Err()
	}
}

func (x *Index) Get(ctx context.Context, name string) ([]byte, error) {
	r := x.current()
	if r == nil {
		return nil, ErrNoGame
	}
	ready, err := r.wait(name)
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return r.get(name)
}

func (x *Index) Peek(name string) ([]byte, error) {
	r := x.current()
	if r == nil {
		return nil, ErrNoGame
	}
	ready, err := r.wait(name)
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
		return r.get(name)
	default:
		return nil, ErrBuilding
	}
}

func (x *Index) open() (fs.FS, error) {
	if x == nil || x.source == nil {
		return nil, ErrNoGame
	}
	fsys, err := x.source()
	if err != nil || fsys == nil {
		return nil, ErrNoGame
	}
	return fsys, nil
}

func (x *Index) current() *run {
	if x == nil {
		return nil
	}
	if x.static != nil {
		return x.static
	}
	fsys, err := x.open()
	x.mu.Lock()
	defer x.mu.Unlock()
	if err != nil {
		x.stopLocked()
		return nil
	}
	id := Identity(fsys)
	if x.run != nil && x.run.id == id {
		return x.run
	}
	x.stopLocked()
	return x.startLocked(fsys, false)
}

func (x *Index) stopLocked() {
	if x.run != nil {
		x.run.cancel()
		x.run = nil
	}
}

func (x *Index) startLocked(fsys fs.FS, fresh bool) *run {
	id := Identity(fsys)
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{
		id: id, fsys: fsys, gameVersion: readGameVersion(fsys), cancel: cancel,
		state: "building", total: len(x.datasets),
		data: map[string][]byte{}, errs: map[string]error{}, ready: map[string]chan struct{}{},
		values: map[any]*value{}, over: make(chan struct{}),
	}
	if x.root != "" {
		r.dir = filepath.Join(x.root, shortHash(id))
	}
	for _, d := range x.datasets {
		r.ready[d.Name] = make(chan struct{})
	}
	if len(x.datasets) > 0 {
		r.step = x.datasets[0].Name
	}
	x.run = r
	go x.build(ctx, r, fresh)
	return r
}

type manifest struct {
	Format      int            `json:"format"`
	Identity    string         `json:"identity"`
	GameVersion string         `json:"game_version"`
	BuiltAt     time.Time      `json:"built_at"`
	Datasets    map[string]int `json:"datasets"`

	Millis map[string]int64 `json:"build_ms,omitempty"`
}

func (x *Index) build(ctx context.Context, r *run, fresh bool) {
	stored := manifest{Datasets: map[string]int{}}
	if r.dir != "" && !fresh {
		if raw, err := os.ReadFile(filepath.Join(r.dir, "manifest.json")); err == nil {
			var m manifest
			if json.Unmarshal(raw, &m) == nil && m.Format == FormatVersion && m.Identity == r.id && m.Datasets != nil {
				stored = m
			}
		}
	}
	if r.dir != "" {
		if err := os.MkdirAll(r.dir, 0o755); err != nil {
			r.dir = ""
		}
	}
	var mu sync.Mutex
	rebuilt := map[string]bool{}
	built := manifest{Format: FormatVersion, Identity: r.id, GameVersion: r.gameVersion, Datasets: map[string]int{}, Millis: map[string]int64{}}
	anyBuilt := false

	sem := make(chan struct{}, min(buildWorkers, runtime.GOMAXPROCS(0)))
	var wg sync.WaitGroup
	for _, d := range x.datasets {
		wg.Add(1)
		go func(d Dataset) {
			defer wg.Done()
			for _, dep := range d.Deps {
				select {
				case <-r.ready[dep]:
				case <-ctx.Done():
					return
				}
			}
			if ctx.Err() != nil {
				return
			}
			storedVersion, ok := stored.Datasets[d.Name]
			reuse := ok && storedVersion == d.Version && r.dir != ""
			failedDep := ""
			mu.Lock()
			for _, dep := range d.Deps {
				if rebuilt[dep] {
					reuse = false
				}
			}
			mu.Unlock()
			for _, dep := range d.Deps {
				if r.errOf(dep) != nil {
					failedDep = dep
				}
			}
			if reuse {
				if _, err := os.Stat(filepath.Join(r.dir, d.Name+".json")); err != nil {
					reuse = false
				}
			}
			if reuse {
				mu.Lock()
				built.Datasets[d.Name] = d.Version
				if ms, ok := stored.Millis[d.Name]; ok {
					built.Millis[d.Name] = ms
				}
				mu.Unlock()
				r.finish(d.Name, nil, nil)
				return
			}
			if failedDep != "" {
				r.finish(d.Name, nil, fmt.Errorf("%s: needs %s, which failed", d.Name, failedDep))
				return
			}
			deps := map[string][]byte{}
			for _, dep := range d.Deps {
				data, err := r.get(dep)
				if err != nil {
					r.finish(d.Name, nil, fmt.Errorf("%s: %w", d.Name, err))
					return
				}
				deps[dep] = data
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			r.setStep(d.Name)
			started := time.Now()
			data, err := safeBuild(ctx, d, &Input{FS: r.fsys, GameVersion: r.gameVersion, deps: deps})
			elapsed := time.Since(started).Milliseconds()
			<-sem
			if err != nil {
				if ctx.Err() == nil {
					r.finish(d.Name, nil, fmt.Errorf("%s: %w", d.Name, err))
				}
				return
			}
			if r.dir != "" {
				if err := writeAtomic(filepath.Join(r.dir, d.Name+".json"), data); err != nil {
					r.finish(d.Name, nil, fmt.Errorf("%s: %w", d.Name, err))
					return
				}
			}
			mu.Lock()
			built.Millis[d.Name] = elapsed
			if r.dir != "" {
				built.Datasets[d.Name] = d.Version
			}
			rebuilt[d.Name] = true
			anyBuilt = true
			mu.Unlock()
			r.finish(d.Name, data, nil)
		}(d)
	}
	wg.Wait()
	if ctx.Err() != nil {
		r.failRest(ctx.Err())
		return
	}
	built.BuiltAt = stored.BuiltAt
	if anyBuilt || built.BuiltAt.IsZero() {
		built.BuiltAt = time.Now().UTC().Truncate(time.Second)
	}
	if r.dir != "" {
		if raw, err := json.MarshalIndent(built, "", " "); err == nil {
			_ = writeAtomic(filepath.Join(r.dir, "manifest.json"), raw)
		}
		x.prune(r.dir)
	}
	r.complete(built.BuiltAt)
}

func safeBuild(ctx context.Context, d Dataset, in *Input) (data []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("builder panicked: %v", p)
		}
	}()
	data, err = d.Build(ctx, in)
	if err == nil && data == nil {
		err = errors.New("builder returned no data")
	}
	return data, err
}

func (x *Index) prune(keep string) {
	entries, err := os.ReadDir(x.root)
	if err != nil {
		return
	}
	type dir struct {
		path string
		at   time.Time
	}
	var dirs []dir
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(x.root, entry.Name())
		if path == keep {
			continue
		}
		info, err := os.Stat(filepath.Join(path, "manifest.json"))
		at := time.Time{}
		if err == nil {
			at = info.ModTime()
		}
		dirs = append(dirs, dir{path, at})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].at.After(dirs[j].at) })
	for i, d := range dirs {
		if i >= 2 {
			_ = os.RemoveAll(d.path)
		}
	}
}

func (r *run) setStep(name string) {
	r.mu.Lock()
	r.step = name
	r.mu.Unlock()
}

func (r *run) finish(name string, data []byte, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil && !errors.Is(err, ErrFailed) {
		err = fmt.Errorf("%w: %w", ErrFailed, err)
	}
	if err != nil {
		r.errs[name] = err
		if r.err == nil {
			r.err = err
		}
	} else if data != nil {
		r.data[name] = data
	}
	r.done++
	if ch := r.ready[name]; ch != nil {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}

func (r *run) failRest(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer closeOnce(r.over)
	r.state = "failed"
	if r.err == nil {
		r.err = err
	}
	for name, ch := range r.ready {
		select {
		case <-ch:
		default:
			r.errs[name] = err
			close(ch)
		}
	}
}

func (r *run) complete(at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer closeOnce(r.over)
	r.step = ""
	r.builtAt = at
	if r.err != nil {
		r.state = "failed"
	} else {
		r.state = "ready"
	}
}

func (r *run) errOf(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errs[name]
}

func (r *run) wait(name string) (chan struct{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch, ok := r.ready[name]
	if !ok {
		if r.static {
			return nil, ErrNoGame
		}
		return nil, fmt.Errorf("gameindex: unknown dataset %q", name)
	}
	return ch, nil
}

func (r *run) get(name string) ([]byte, error) {
	r.mu.Lock()
	if err := r.errs[name]; err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if data, ok := r.data[name]; ok {
		r.mu.Unlock()
		return data, nil
	}
	dir := r.dir
	r.mu.Unlock()
	if dir == "" {
		return nil, fmt.Errorf("gameindex: dataset %q is missing", name)
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return nil, fmt.Errorf("gameindex: %w", err)
	}
	r.mu.Lock()
	r.data[name] = data
	r.mu.Unlock()
	return data, nil
}

func (r *run) status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := Status{State: r.state, GameVersion: r.gameVersion, Step: r.step, Done: r.done, Total: r.total}
	if r.err != nil {
		s.Error = r.err.Error()
	}
	if !r.builtAt.IsZero() {
		s.BuiltAt = r.builtAt.Format(time.RFC3339)
	}
	return s
}

func Value[T any](ctx context.Context, x *Index, key any, name string, decode func([]byte) (T, error)) (T, error) {
	var zero T
	r := x.current()
	if r == nil {
		return zero, ErrNoGame
	}
	data, err := x.Get(ctx, name)
	if err != nil {
		return zero, err
	}
	r.mu.Lock()
	v := r.values[key]
	if v == nil {
		v = &value{}
		r.values[key] = v
	}
	r.mu.Unlock()
	v.once.Do(func() { v.v, v.err = decode(data) })
	if v.err != nil {
		return zero, v.err
	}
	out, _ := v.v.(T)
	return out, nil
}

func Identity(fsys fs.FS) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%T", fsys))
	v := reflect.ValueOf(fsys)
	switch v.Kind() {
	case reflect.String:
		b.WriteString(":" + v.String())
	case reflect.Map, reflect.Pointer, reflect.Slice, reflect.Func:
		b.WriteString(fmt.Sprintf("@%x", v.Pointer()))
	default:
		b.WriteString(fmt.Sprintf("%v", fsys))
	}
	if info, err := fs.Stat(fsys, "launcher-settings.json"); err == nil {
		b.WriteString(fmt.Sprintf("|%d|%d", info.Size(), info.ModTime().UnixNano()))
	}
	for _, group := range []string{"dlc", "integrated_dlc"} {
		entries, _ := fs.ReadDir(fsys, group)
		for _, entry := range entries {
			if entry.IsDir() {
				b.WriteString("|" + group + "/" + entry.Name())
			}
		}
	}
	return b.String()
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func readGameVersion(fsys fs.FS) string {
	raw, err := fs.ReadFile(fsys, "launcher-settings.json")
	if err != nil {
		return ""
	}
	var settings struct {
		RawVersion string `json:"rawVersion"`
	}
	if json.Unmarshal(raw, &settings) != nil {
		return ""
	}
	return settings.RawVersion
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

func closeOnce(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}
