package form

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// FieldParser allows callers to implement custom form parsing for a particular field
type FieldParser interface {
	ParseField(key string, vals []string) error
}

// Parse calls Parser{}.Parse(
// Parser performs the parsing of the form values. It is used to specify options that
// alter the method of parsing
type Decoder struct {
	strictCase bool
	// Recurse allows sub structs to be populated. Leave nil if you do not want sub keys to be parsed.
	// If Recurse is nil and a sub struct/map is found then it is ignored.
	recurse DecodeSubKeyFunc
}

func NewDecoder() *Decoder {
	return &Decoder{strictCase: true}
}

func (d *Decoder) StrictCase(b bool) {
	d.strictCase = b
}

func (d *Decoder) Recurse(f DecodeSubKeyFunc) {
	d.recurse = f
}

// Parse parses the form values into the supplied variable based on the parsers options.
// The supplied value must be either a non-nil pointer to a struct or a map.
func (p *Decoder) Decode(src map[string][]string, dst interface{}) error {
	entries, err := buildMap(src, !p.strictCase, p.recurse != nil)
	if err != nil {
		return err
	}

	vals := buildVals(src, !p.strictCase, p.recurse)
	return setMap(vals, entries)
}

func buildVals(vals url.Values, toLower bool, recurse DecodeSubKeyFunc) map[string]*formLayer {
	flatVals := make(map[string]*formLayer)
	for k, e := range vals {
		key := k
		if toLower {
			key = strings.ToLower(k)
		}

		if prev := flatVals[key]; prev == nil {
			// No subkeys yet
			flatVals[key] = &formLayer{val: &formVal{key: k, vals: e}}
		} else {
			// There are multiple entries for the same key with different cases. Just sacrifice some
			// of error messaging
			prev.val.vals = append(prev.val.vals, e...)
		}
	} // for

	if recurse == nil {
		return flatVals
	}

	// Expand keys for subkey access.

	recVals := make(map[string]*formLayer)

	for key, layer := range flatVals {
		keys := recurse(key)
		if len(keys) == 0 {
			panic("invalid number of keys from recurse function")
		}

		currMap := recVals
		for i, subKey := range keys {
			currEntry := currMap[subKey]

			// Initialise the layer if we have to
			if currEntry == nil {
				currEntry = &formLayer{}
				currMap[subKey] = currEntry
			}

			// Are we at the end? If so then set the vals
			if i == len(keys)-1 {
				// TODO: Return error on duplicate?
				if currEntry.val == nil {
					currEntry.val = layer.val
				} else {
					currEntry.val.vals = append(currEntry.val.vals, layer.val.vals...)
				}
			} else { // There are sub keys so add to the map
				if currEntry.subVals == nil {
					currEntry.subVals = make(map[string]*formLayer)
				}

				currMap = currEntry.subVals
			}
		}
	}

	return recVals
}

type formLayer struct {
	val     *formVal
	subVals map[string]*formLayer
}

type formVal struct {
	vals []string
	// preserved original key for errors.
	key string
}

func setMap(vals map[string]*formLayer, entries map[string]*formEntry) error {
	for k, layer := range vals {
		entry, ok := entries[k]
		if !ok {
			// TODO: Error?
			continue
		}

		if layer.val != nil {
			// If nil we can now set the field.
			if entry.field.Value.Kind() == reflect.Ptr && entry.field.Value.IsNil() && entry.field.Value.CanSet() {
				if entry.field.Value.IsNil() && entry.field.Value.CanSet() {
					entry.field.Value.Set(reflect.New(entry.field.Value.Type().Elem()))
				} // if
			} // if

			if fieldParser, ok := getFieldParser(entry.field.Value); ok {
				if err := fieldParser.ParseField(k, layer.val.vals); err != nil {
					return &FieldParseError{Field: entry.field.Name, Err: err}
				} // if

				return nil
			} // if

			// We have a few kinds
			v := baseElem(entry.field.Value)
			kind := v.Kind()
			if !v.CanSet() {
				return nil
			}

			switch kind {
			case reflect.Bool:
				b, err := parseBool(layer.val.vals)
				if err != nil {
					return &FieldParseError{Field: entry.field.Name, Err: err}
				}

				v.SetBool(b)

			case reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64:
				i, err := parseInt(layer.val.vals)
				if err != nil {
					return &FieldParseError{Field: entry.field.Name, Err: err}
				}

				v.SetInt(i)

			case reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64:
				i, err := parseUint(layer.val.vals)
				if err != nil {
					return &FieldParseError{Field: entry.field.Name, Err: err}
				}

				v.SetUint(i)

			case reflect.Slice:
				elem := v.Type().Elem()
				if elem.Kind() != reflect.String {
					return &FieldTypeError{Field: entry.field.Name, Type: entry.field.Value.Type()}
				}

				// Copy all of the elements into the new string type
				// The inner string type has been aliased. We need to convert each element
				set := reflect.MakeSlice(v.Type(), len(layer.val.vals), len(layer.val.vals))

				for i := range layer.val.vals {
					set.Index(i).SetString(layer.val.vals[i])
				}

				v.Set(set)

			case reflect.String:
				s, err := parseString(layer.val.vals)
				if err != nil {
					return &FieldParseError{Field: entry.field.Name, Err: err}
				}

				v.SetString(s)

			default:
				return &FieldTypeError{Field: entry.field.Name, Type: entry.field.Value.Type()}
			}
		}

		if layer.subVals != nil && entry.subEntries != nil {
			setMap(layer.subVals, entry.subEntries)
		}
	}

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

type formEntry struct {
	field      *Field
	subEntries map[string]*formEntry
}

func buildMap(v interface{}, toLower, recurse bool) (map[string]*formEntry, error) {
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

	entries := make(map[string]*formEntry)
	if err := addMapEntries(entries, ele, toLower, recurse); err != nil {
		return nil, err
	}

	return entries, nil
} // buildMap

func addMapEntries(entries map[string]*formEntry, ele reflect.Value, toLower, recurse bool) error {
	eleType := ele.Type()
	nFields := ele.NumField()

	for i := 0; i < nFields; i++ {
		entry := ele.Field(i)
		entryType := eleType.Field(i)

		isUnexported := entryType.PkgPath != ""
		if isUnexported {
			continue
		}

		name := entryType.Name
		if tag := entryType.Tag.Get("form"); tag != "" {
			name = tag
		}

		keyName := name
		if toLower {
			keyName = strings.ToLower(name)
		}

		if _, ok := entries[keyName]; ok {
			return &DuplicateFieldError{Field: name}
		}

		if !entry.CanSet() {
			continue
		}

		if !entry.CanInterface() {
			continue
		}

		// Add anonymous structs values at the same level as the current
		if entryType.Type.Kind() == reflect.Struct && entryType.Anonymous {
			if err := addMapEntries(entries, entry, toLower, recurse); err != nil {
				return err
			}

			continue
		}

		fEntry := &formEntry{
			field: &Field{
				Value: entry,
				Name:  name,
			},
		}

		if recurse && entryType.Type.Kind() == reflect.Struct {
			fEntry.subEntries = make(map[string]*formEntry)

			if err := addMapEntries(fEntry.subEntries, entry, toLower, recurse); err != nil {
				return err
			}
		}

		entries[keyName] = fEntry
	} // for

	return nil
}

// Wrong type passed into Parse
type UsageTypeError struct {
	Type reflect.Type
}

func (e *UsageTypeError) Error() string {
	if e.Type == nil {
		return buildErrorMessage("Parse", "nil")
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
