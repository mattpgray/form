package form

import (
	"fmt"
	"log"
	"net/url"
	"reflect"
)

var defaultParser = &Parser{}

func Parse(vals url.Values, v interface{}) error {
	return defaultParser.Parse(vals, v)
}

type Parser struct {
	CaseInsensitive bool
	AllowExtra      bool
}

func (p *Parser) Parse(vals url.Values, v interface{}) error {

	entries := buildMap(v)
	if entries == nil {
		return &UsageTypeError{Type: reflect.TypeOf(v)}
	} // if

	return nil
}

func buildMap(v interface{}) map[string]interface{} {
	if entries, ok := v.(map[string]interface{}); ok {
		// TODO: Check that each entry can be set.
		return entries
	} // if

	// Must be a pointer
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	} // if

	// Must be a struct
	ele := rv.Elem()
	if ele.Kind() != reflect.Struct {
		return nil
	} // if

	eleType := ele.Type()

	nFields := ele.NumField()
	entries := make(map[string]interface{}, nFields)
	for i := 0; i < nFields; i++ {
		// entry := ele.Field(i)
		entryType := eleType.Field(i)

		isUnexported := entryType.PkgPath != ""
		// TODO: Handle anonymous structs.
		if isUnexported {
			continue
		} // if

		name := entryType.Name

		tag, ok := entryType.Tag.Lookup("form")

		log.Printf("name: %q, tag %q, ok %t", name, tag, ok)
	}

	return entries
}

// Wrong type passed into Parse
type UsageTypeError struct {
	Fn   string
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

func buildErrorMessage(fn, msg string) string {
	return fmt.Sprintf("go-form:%s: %s", fn, msg)
} // buildErrorMessage
