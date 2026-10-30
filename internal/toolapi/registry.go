package toolapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type Server struct {
	tools      map[string]*registeredTool
	middleware []Middleware
}

type registeredTool struct {
	tool    *Tool
	handler func(context.Context, *CallToolRequest) (*CallToolResult, error)
}

func NewServer() *Server { return &Server{tools: map[string]*registeredTool{}} }

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type CallToolParamsRaw struct {
	Name      string
	Arguments json.RawMessage
}

type Request interface{ GetParams() any }

type CallToolRequest struct{ Params *CallToolParamsRaw }

func (r *CallToolRequest) GetParams() any { return r.Params }

type ListToolsRequest struct{}

func (*ListToolsRequest) GetParams() any { return nil }

type Result any

type ListToolsResult struct{ Tools []*Tool }

type CallToolResult struct {
	IsError           bool
	Content           []Content
	StructuredContent any

	err error
}

func (r *CallToolResult) SetError(err error) {
	r.IsError = true
	r.Content = []Content{&TextContent{Text: err.Error()}}
	r.err = err
}

type Content interface{ isContent() }

type TextContent struct{ Text string }

type ImageContent struct {
	Data     []byte
	MIMEType string
}

func (*TextContent) isContent()  {}
func (*ImageContent) isContent() {}

type MethodHandler func(ctx context.Context, method string, req Request) (Result, error)

type Middleware func(MethodHandler) MethodHandler

func (s *Server) AddReceivingMiddleware(middleware ...Middleware) {
	s.middleware = append(s.middleware, middleware...)
}

var errInvalidArguments = errors.New("invalid arguments")

func AddTool[In, Out any](s *Server, t *Tool, h func(context.Context, *CallToolRequest, In) (*CallToolResult, Out, error)) {
	if _, taken := s.tools[t.Name]; taken {
		panic("toolapi: tool registered twice: " + t.Name)
	}
	inType := reflect.TypeFor[In]()
	tool := *t
	tool.InputSchema = schemaFor(inType, map[reflect.Type]bool{})
	s.tools[t.Name] = &registeredTool{tool: &tool, handler: func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		var in In
		if err := decodeArguments(req.Params.Arguments, inType, &in); err != nil {
			result := &CallToolResult{}
			result.SetError(fmt.Errorf("%w: %v", errInvalidArguments, err))
			return result, nil
		}
		res, out, err := h(ctx, req, in)
		if err != nil {
			result := &CallToolResult{}
			result.SetError(err)
			return result, nil
		}
		if res == nil {
			res = &CallToolResult{}
		}
		if value := reflect.ValueOf(out); value.IsValid() {
			switch {
			case value.Kind() == reflect.Pointer && value.IsNil():
				res.StructuredContent = reflect.New(value.Type().Elem()).Interface()
			case (value.Kind() == reflect.Map || value.Kind() == reflect.Interface || value.Kind() == reflect.Slice) && value.IsNil():
			default:
				res.StructuredContent = out
			}
		}
		return res, nil
	}}
}

func (s *Server) handle(ctx context.Context, method string, req Request) (Result, error) {
	switch method {
	case "tools/list":
		names := make([]string, 0, len(s.tools))
		for name := range s.tools {
			names = append(names, name)
		}
		sort.Strings(names)
		list := &ListToolsResult{Tools: make([]*Tool, 0, len(names))}
		for _, name := range names {
			copyTool := *s.tools[name].tool
			list.Tools = append(list.Tools, &copyTool)
		}
		return list, nil
	case "tools/call":
		call, _ := req.(*CallToolRequest)
		if call == nil || call.Params == nil {
			return nil, errors.New("no tool named")
		}
		registered, ok := s.tools[call.Params.Name]
		if !ok {
			return nil, &UnknownToolError{Name: call.Params.Name}
		}
		return registered.handler(ctx, call)
	}
	return nil, fmt.Errorf("unknown method %q", method)
}

func (s *Server) chain() MethodHandler {
	handler := MethodHandler(s.handle)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}
	return handler
}

type UnknownToolError struct{ Name string }

func (e *UnknownToolError) Error() string { return fmt.Sprintf("unknown tool %q", e.Name) }

func (s *Server) ListTools(ctx context.Context) ([]*Tool, error) {
	result, err := s.chain()(ctx, "tools/list", &ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	list, _ := result.(*ListToolsResult)
	if list == nil {
		return nil, errors.New("tool list unavailable")
	}
	return list.Tools, nil
}

func (s *Server) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	if len(arguments) == 0 {
		arguments = json.RawMessage("{}")
	}
	result, err := s.chain()(ctx, "tools/call", &CallToolRequest{Params: &CallToolParamsRaw{Name: name, Arguments: arguments}})
	var unknown *UnknownToolError
	if errors.As(err, &unknown) {
		return nil, err
	}
	if err != nil {
		call := &CallToolResult{}
		call.SetError(err)
		return call, nil
	}
	call, _ := result.(*CallToolResult)
	if call == nil {
		return nil, errors.New("tool returned no result")
	}
	return call, nil
}

var rawMessageType = reflect.TypeFor[json.RawMessage]()

func schemaFor(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == rawMessageType {
		return map[string]any{}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaFor(t.Elem(), seen)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(t.Elem(), seen)}
	case reflect.Struct:
		if seen[t] {
			return map[string]any{"type": "object"}
		}
		seen[t] = true
		defer delete(seen, t)
		properties := map[string]any{}
		required := []string{}
		for _, field := range structFields(t) {
			property := schemaFor(field.typ, seen)
			if field.description != "" {
				property["description"] = field.description
			}
			properties[field.name] = property
			if field.required {
				required = append(required, field.name)
			}
		}
		schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	return map[string]any{}
}

type jsonField struct {
	name        string
	typ         reflect.Type
	required    bool
	description string
}

func structFields(t reflect.Type) []jsonField {
	var out []jsonField
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, options, _ := strings.Cut(tag, ",")
		if field.Anonymous && name == "" {
			embedded := field.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				out = append(out, structFields(embedded)...)
				continue
			}
		}
		if !field.IsExported() {
			continue
		}
		if name == "" {
			name = field.Name
		}
		optional := false
		for _, option := range strings.Split(options, ",") {
			if option == "omitempty" || option == "omitzero" {
				optional = true
			}
		}
		out = append(out, jsonField{name: name, typ: field.Type, required: !optional, description: field.Tag.Get("jsonschema")})
	}
	return out
}

func decodeArguments(raw json.RawMessage, t reflect.Type, target any) error {
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("arguments are not JSON: %v", err)
	}
	if err := checkValue(value, t, "arguments", map[reflect.Type]bool{}); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return fmt.Errorf("%s: expected %s, got a JSON %s", typeErr.Field, typeErr.Type, typeErr.Value)
		}
		return err
	}
	return nil
}

func checkValue(value any, t reflect.Type, path string, seen map[reflect.Type]bool) error {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable, t = true, t.Elem()
	}
	if value == nil {
		switch t.Kind() {
		case reflect.Slice, reflect.Map, reflect.Interface:
			return nil
		}
		if nullable || t == rawMessageType {
			return nil
		}
		return fmt.Errorf("%s must not be null", path)
	}
	if t == rawMessageType {
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		if seen[t] {
			return nil
		}
		seen[t] = true
		defer delete(seen, t)
		fields := structFields(t)
		known := make(map[string]jsonField, len(fields))
		for _, field := range fields {
			known[field.name] = field
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			field, ok := known[key]
			if !ok {
				return fmt.Errorf("%s has an unknown property %q", path, key)
			}
			if err := checkValue(object[key], field.typ, path+"."+key, seen); err != nil {
				return err
			}
		}
		for _, field := range fields {
			if _, present := object[field.name]; field.required && !present {
				return fmt.Errorf("%s is missing the required property %q", path, field.name)
			}
		}
	case reflect.Slice, reflect.Array:
		items, ok := value.([]any)
		if !ok {
			if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
				return nil
			}
			return fmt.Errorf("%s must be an array", path)
		}
		for index, item := range items {
			if err := checkValue(item, t.Elem(), fmt.Sprintf("%s[%d]", path, index), seen); err != nil {
				return err
			}
		}
	case reflect.Map:
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		for key, item := range object {
			if err := checkValue(item, t.Elem(), path+"."+key, seen); err != nil {
				return err
			}
		}
	}
	return nil
}
