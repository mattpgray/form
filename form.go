package form

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type FieldParser interface {
	ParseField(vals []string) error
}

var defaultParser = &Parser{}

func Parse(vals url.Values, v interface{}) error {
	return defaultParser.Parse(vals, v)
}

type Parser struct {
	CaseInsensitive bool
	AllowExtra      bool
}

func (p *Parser) Parse(vals url.Values, v interface{}) error {
	entries, err := buildMap(v)
	if err != nil {
		return err
	} // if

	if p.CaseInsensitive {
		// Change all struct tags to lower case for fast searching
		entriesLower := make(map[string]*Field, len(entries))
		for k, entry := range entries {
			kLower := strings.ToLower(k)
			if _, ok := entriesLower[kLower]; ok {
				return &DuplicateFieldError{Field: kLower}
			} // if

			entriesLower[kLower] = entry
		} // for

		entries = entriesLower

		// Also flatten vals from multiple cases
		valsLower := url.Values{}
		for k, e := range vals {
			kLower := strings.ToLower(k)

			if !p.AllowExtra {
				// Check here to preserve the initial case of the incorrect field
				if _, ok := entries[kLower]; !ok {
					return &UnexpectedFieldError{Field: k, Vals: e}
				}
			}

			valsLower[kLower] = append(valsLower[kLower], e...)
		} // for

		vals = valsLower
	} else if !p.AllowExtra {
		// Check that we expected every value before beginning to parse values.
		// This helps avoid partially populating the result.
		for k, e := range vals {
			if _, ok := entries[k]; !ok {
				return &UnexpectedFieldError{Field: k, Vals: e}
			}
		} // for
	}

	return setMap(vals, entries)
}

func setMap(vals url.Values, entries map[string]*Field) error {
	for k, val := range vals {
		// TODO: Preserve inital type and some tag options
		field, ok := entries[k]
		if !ok {
			// this is guarded and should never hapen
			panic("duplicate field found after checking: " + k)
		}

		// If nil we can now set the field.
		if field.Value.Kind() == reflect.Ptr && field.Value.IsNil() && field.Value.CanSet() {
			if field.Value.IsNil() && field.Value.CanSet() {
				field.Value.Set(reflect.New(field.Value.Type().Elem()))
			} // if
		} // if

		if fieldParser, ok := getFieldParser(field.Value); ok {
			if err := fieldParser.ParseField(val); err != nil {
				return &FieldParseError{Field: field.Name, Err: err}
			} // if

			continue
		} // if

		// We have a few kinds
		v := baseElem(field.Value)
		kind := v.Kind()
		if !v.CanSet() {
			continue
		}

		switch kind {
		case reflect.Bool:
			b, err := parseBool(val)
			if err != nil {
				return &FieldParseError{Field: field.Name, Err: err}
			}

			v.SetBool(b)

		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			i, err := parseInt(val)
			if err != nil {
				return &FieldParseError{Field: field.Name, Err: err}
			}

			v.SetInt(i)

		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			i, err := parseUint(val)
			if err != nil {
				return &FieldParseError{Field: field.Name, Err: err}
			}

			v.SetUint(i)

		case reflect.Slice:
			if v.Elem().Kind() != reflect.String {
				return &FieldTypeError{Field: field.Name, Type: field.Value.Type()}
			}

			set := reflect.ValueOf(val)
			if set.Type() != v.Type() {
				set = set.Convert(v.Type())
			}

			v.Set(set)

		case reflect.String:
			s, err := parseString(val)
			if err != nil {
				return &FieldParseError{Field: field.Name, Err: err}
			}

			v.SetString(s)

		default:
			return &FieldTypeError{Field: field.Name, Type: field.Value.Type()}
		}
	} // for

	return nil
}

type Field struct {
	Value reflect.Value
	Name  string
}

func getFieldParser(v reflect.Value) (FieldParser, bool) {
	if fieldParser, ok := getFieldParserOnce(v); ok {
		return fieldParser, true
	}

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		// Get the elem
		// Does this even work?
		if fieldParser, ok := getFieldParserOnce(v.Elem()); ok {
			return fieldParser, true
		}
	}

	// Check the pointer in all cases.
	if v.CanAddr() {
		if fieldParser, ok := getFieldParserOnce(v.Addr()); ok {
			return fieldParser, true
		}
	}

	return nil, false
} // getFieldParser

func getFieldParserOnce(v reflect.Value) (FieldParser, bool) {
	if v.CanInterface() {
		if fieldParser, ok := v.Interface().(FieldParser); ok {
			return fieldParser, true
		}
	} // if

	return nil, false
} // getFieldParserOnce

func baseElem(v reflect.Value) reflect.Value {
	// TODO guard against infinite recursion ?
	for {
		// Check the pointer in all cases.
		if v.Kind() != reflect.Ptr {
			return v
		}

		v = v.Elem()
	}
}

func buildMap(v interface{}) (map[string]*Field, error) {
	if entries, ok := v.(map[string]interface{}); ok {
		e := make(map[string]*Field, len(entries))
		for k, v := range entries {
			e[k] = &Field{
				Value: reflect.ValueOf(v),
				Name:  k,
			}
		}
		// TODO: Check that each entry can be set.
		return e, nil
	} // if

	// Must be a pointer
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil, &UsageTypeError{Type: reflect.TypeOf(v)}
	} // if

	// Must be a struct
	ele := rv.Elem()
	if ele.Kind() != reflect.Struct {
		return nil, &UsageTypeError{Type: reflect.TypeOf(v)}
	} // if

	eleType := ele.Type()

	nFields := ele.NumField()
	entries := make(map[string]*Field, nFields)
	for i := 0; i < nFields; i++ {
		entry := ele.Field(i)
		entryType := eleType.Field(i)

		isUnexported := entryType.PkgPath != ""
		// TODO: Handle anonymous structs.
		if isUnexported {
			continue
		}

		name := entryType.Name
		if tag := entryType.Tag.Get("form"); tag != "" {
			name = tag
		}

		if _, ok := entries[name]; ok {
			return nil, &DuplicateFieldError{Field: name}
		}

		if !entry.CanSet() {
			continue
		}

		if !entry.CanInterface() {
			continue
		}

		entries[name] = &Field{
			Value: entry,
			Name:  name,
		}
	} // for

	return entries, nil
} // buildMap

// Wrong type passed into Parse
type UsageTypeError struct {
	Type reflect.Type
}

func (e *UsageTypeError) Error() string {
	if e.Type == nil {
		buildErrorMessage("Parse", "nil")
	}

	if e.Type.Kind() != reflect.Ptr {
		return buildErrorMessage("Parse", "non-pointer "+e.Type.String())
	}

	return buildErrorMessage("Parse", "nil "+e.Type.String())
}

type FieldTypeError struct {
	Field string
	Type  reflect.Type
}

func (e *FieldTypeError) Error() string {
	return buildErrorMessage("Parse", "invalid type "+e.Type.String()+" in field "+e.Field)
}

type DuplicateFieldError struct {
	Field string
}

func (e *DuplicateFieldError) Error() string {
	return buildErrorMessage("Parse", "duplicate field "+e.Field)
}

type UnexpectedFieldError struct {
	Field string
	Vals  []string
}

func (e *UnexpectedFieldError) Error() string {
	var msg string
	switch len(e.Vals) {
	case 0:
		msg = fmt.Sprintf("unexpected field %q", e.Field)

	case 1:
		msg = fmt.Sprintf("unexpected field %q with value %q", e.Field, e.Vals[0])

	default:
		msg = fmt.Sprintf("unexpected field %q with %d values", e.Field, len(e.Vals[0]))
	}

	return buildErrorMessage("Parse", msg)
}

type UnexpectedValueError struct {
	Value string
}

func (e *UnexpectedValueError) Error() string {
	return buildErrorMessage("Parse", fmt.Sprintf("unexpected value %v", e.Value))
}

type UnexpectedValuesError struct {
	Values []string
}

func (e *UnexpectedValuesError) Error() string {
	return buildErrorMessage("Parse", fmt.Sprintf("unexpected values %v", e.Values))
}

type FieldParseError struct {
	Field string
	Err   error
}

func (e *FieldParseError) Error() string {
	return buildErrorMessage("Parse", fmt.Sprintf("error parsing field %q: %q", e.Field, e.Err))
}

func (e *FieldParseError) Unwrap() error {
	return e.Err
}

func buildErrorMessage(fn, msg string) string {
	return "go-form:" + fn + ": " + msg
}

func parseBool(vals []string) (bool, error) {
	s, err := parseString(vals)
	if err != nil {
		return false, err
	} // if

	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	return false, &UnexpectedValueError{s}
}

func parseInt(vals []string) (int64, error) {
	s, err := parseString(vals)
	if err != nil {
		return 0, err
	} // if

	i, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, &UnexpectedValueError{s}
	} // if

	return i, nil
}

func parseUint(vals []string) (uint64, error) {
	s, err := parseString(vals)
	if err != nil {
		return 0, err
	} // if

	u, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return 0, &UnexpectedValueError{s}
	} // if

	return u, nil
}

func parseString(vals []string) (string, error) {
	if len(vals) == 1 {
		return vals[0], nil
	}

	// We strictly only allow one value when parsing a string.
	// Callers can use []string if they care
	return "", &UnexpectedValuesError{vals}
}
