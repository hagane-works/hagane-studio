package world

type PyDict struct {
	keys []string
	m    map[string]any
}

func NewPyDict() *PyDict { return &PyDict{m: map[string]any{}} }

func (d *PyDict) Set(k string, v any) {
	if _, ok := d.m[k]; !ok {
		d.keys = append(d.keys, k)
	}
	d.m[k] = v
}

func (d *PyDict) Get(k string) (any, bool) {
	v, ok := d.m[k]
	return v, ok
}

func (d *PyDict) MustGet(k string) any { return d.m[k] }

func (d *PyDict) Keys() []string { return d.keys }

func (d *PyDict) Len() int { return len(d.keys) }

func (d *PyDict) appendTo(k string, v any) {
	lst, _ := d.m[k].([]any)
	d.Set(k, append(lst, v))
}

func isEmptyStrValue(v any) bool {
	s, ok := v.(string)
	return ok && s == ""
}

func (d *PyDict) Accumulate(key string, piece any) {
	existing, present := d.Get(key)
	if !present || existing == nil || isEmptyStrValue(existing) {
		d.Set(key, piece)
	} else if lst, ok := existing.([]any); ok {
		d.Set(key, append(lst, piece))
	} else {
		d.Set(key, []any{existing, piece})
	}
}
