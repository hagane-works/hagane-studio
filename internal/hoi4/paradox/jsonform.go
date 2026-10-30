package paradox

type CommentJSON struct {
	Text   string `json:"text"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Inline bool   `json:"inline"`
	WS     string `json:"ws"`
}

type PayloadJSON struct {
	Leading  []CommentJSON `json:"leading"`
	Trailing []CommentJSON `json:"trailing"`
}

type ValueJSON struct {
	T      string       `json:"t"`
	V      any          `json:"v"`
	Trivia *PayloadJSON `json:"trivia,omitempty"`
}

type EntryJSON struct {
	Key       *string     `json:"key"`
	Op        *string     `json:"op"`
	Value     ValueJSON   `json:"value"`
	Line      int         `json:"line"`
	Quoted    bool        `json:"quoted"`
	Qualifier *string     `json:"qualifier"`
	Trivia    PayloadJSON `json:"trivia"`
}

type RecoveryJSON struct {
	Code    string         `json:"code"`
	Line    int            `json:"line"`
	Details map[string]any `json:"details"`
}

func CommentsJSON(cs []Comment) []CommentJSON {
	out := make([]CommentJSON, 0, len(cs))
	for _, c := range cs {
		out = append(out, CommentJSON{c.Text, c.Line, c.Column, c.Inline, c.WS})
	}
	return out
}

func ToPayloadJSON(p Payload) PayloadJSON {
	return PayloadJSON{Leading: CommentsJSON(p.Leading), Trailing: CommentsJSON(p.Trailing)}
}

func toValueJSON(e *Entry) ValueJSON {
	if e.Block != nil {
		tr := ToPayloadJSON(e.Block.Trivia)
		return ValueJSON{T: "block", V: EntriesJSON(e.Block.Entries), Trivia: &tr}
	}
	switch e.Scalar.Kind {
	case ScalarInt:
		return ValueJSON{T: "int", V: e.Scalar.StrValue()}
	case ScalarFloat:
		return ValueJSON{T: "float", V: e.Scalar.StrValue()}
	default:
		return ValueJSON{T: "str", V: e.Scalar.Str}
	}
}

func EntriesJSON(es []*Entry) []EntryJSON {
	out := make([]EntryJSON, 0, len(es))
	for _, e := range es {
		out = append(out, EntryJSON{
			Key: e.Key, Op: e.Op, Value: toValueJSON(e), Line: e.Line,
			Quoted: e.Quoted, Qualifier: e.Qualifier, Trivia: ToPayloadJSON(e.Trivia),
		})
	}
	return out
}

func RecoveriesJSON(rs []Recovery) []RecoveryJSON {
	out := make([]RecoveryJSON, 0, len(rs))
	for _, r := range rs {
		out = append(out, RecoveryJSON{Code: r.Code, Line: r.Line, Details: r.Details})
	}
	return out
}
