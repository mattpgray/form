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
		entriesLower := make(map[string]interface{}, len(entries))
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
		// This helpes avoid partially populating the result.
		for k, e := range vals {
			if _, ok := entries[k]; !ok {
				return &UnexpectedFieldError{Field: k, Vals: e}
			}
		} // for
	}

	for k, e := range vals {
		// TODO: Preserve inital type and some tag options
		entry, ok := entries[k]
		if !ok {
			// this is guarded and should never hapen
			panic("duplicate field found after checking: " + k)
		}

		if fieldParser, ok := entry.(FieldParser); ok {
			if err := fieldParser.ParseField(e); err != nil {
				return err
			} // if

			continue
		} // if

		// Common options

		switch entryVal := entry.(type) {
		case *string:
			if len(e) > 0 {
				*entryVal = e[0]
			}

		case *int:
			if len(e) > 0 {
				if i, err := strconv.Atoi(e[0]); err != nil {
					return err
				} else {
					*entryVal = i
				}
			}

		default:
			return &FieldTypeError{Field: k, Type: reflect.TypeOf(e)}
		}
	} // for

	return nil
}

func buildMap(v interface{}) (map[string]interface{}, error) {
	if entries, ok := v.(map[string]interface{}); ok {
		// TODO: Check that each entry can be set.
		return entries, nil
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
	entries := make(map[string]interface{}, nFields)
	for i := 0; i < nFields; i++ {
		entry := ele.Field(i)
		entryType := eleType.Field(i)

		isUnexported := entryType.PkgPath != ""
		// TODO: Handle anonymous structs.
		if isUnexported {
			continue
		} // if

		name := entryType.Name
		if tag := entryType.Tag.Get("form"); tag != "" {
			name = tag
		} // if

		if _, ok := entries[name]; ok {
			return nil, &DuplicateFieldError{Field: name}
		} // if

		if entry.Kind() == reflect.Ptr {
			// If it is a nil pointer then populate it
			if entry.IsNil() {
				if entry.CanSet() {
					continue
				} // if

				entry.Set(reflect.New(entry.Type().Elem()))
			} // if
		} else {
			// Get the address
			if !entry.CanAddr() {
				continue
			} // if

			entry = entry.Addr()
		} // else

		if !entry.CanInterface() {
			continue
		} // if

		entries[name] = entry.Interface()
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

func buildErrorMessage(fn, msg string) string {
	return "go-form:" + fn + ": " + msg
}
