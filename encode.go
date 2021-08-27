package form

import (
	"net/url"
	"reflect"
	"strconv"
)

// FieldParser allows callers to implement custom form parsing for a particular field
type FieldEncoder interface {
	EncodeField() ([]string, error)
}

var defaultEncoder = &Encoder{}

// Parse calls Parser{}.Parse()
func Encode(v interface{}) (url.Values, error) {
	return defaultEncoder.Encode(v)
}

func EncodeString(v interface{}) (string, error) {
	return defaultEncoder.EncodeString(v)
}

// Parser performs the parsing of the form values. It is used to specify options that
// alter the method of parsing
type Encoder struct {
	CaseInsensitive bool
	AllowExtra      bool
	// Recurse allows sub structs to be populated. Leave nil if you do not want sub keys to be parsed.
	// If Recurse is nil and a sub struct/map is found then it is ignored.
	Recurse EncodeSubKeyFunc
}

// Parse parses the form values into the supplied variable based on the parsers options.
// The supplied value must be either a non-nil pointer to a struct or a map.
func (p *Encoder) Encode(v interface{}) (url.Values, error) {
	// TODO: Map

	vals := url.Values{}

	ele := baseElem(reflect.ValueOf(v))
	if err := addURLVals(vals, ele, nil, nil); err != nil {
		return nil, err
	} // if

	return vals, nil
}

func (p *Encoder) EncodeString(v interface{}) (string, error) {
	vals, err := Encode(v)
	if err != nil {
		return "", err
	}

	return vals.Encode(), nil
}

func addURLVals(vals url.Values, ele reflect.Value, prevKeys []string, recurse func(keys []string) string) error {
	eleType := ele.Type()
	nFields := ele.NumField()

	for i := 0; i < nFields; i++ {
		entry := baseElem(ele.Field(i))
		entryType := eleType.Field(i)

		isUnexported := entryType.PkgPath != ""
		if isUnexported {
			continue
		}

		name := entryType.Name
		if tag := entryType.Tag.Get("form"); tag != "" {
			name = tag
		}

		if _, ok := vals[name]; ok {
			return &DuplicateFieldError{Field: name}
		}

		// Add anonymous structs values at the same level as the current
		if entryType.Type.Kind() == reflect.Struct && entryType.Anonymous {
			if err := addURLVals(vals, entry, prevKeys, recurse); err != nil {
				return err
			}
			continue
		}

		// First custom encoding
		if fieldEncoder, ok := getFieldEncoder(entry); ok {
			v, err := fieldEncoder.EncodeField()
			// TODO: Meta info on error
			if err != nil {
				return err
			}

			vals[name] = v
		} else {
			if recurse != nil && entryType.Type.Kind() == reflect.Struct {
				nextKeys := append(prevKeys, name)
				if err := addURLVals(vals, entry, nextKeys, recurse); err != nil {
					return err
				}

				continue
			}

			switch entry.Type().Kind() {
			case reflect.Bool:
				if entry.Bool() {
					vals[name] = []string{"true"}
				} else {
					vals[name] = []string{"false"}
				}

			case reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64:
				vals[name] = []string{strconv.FormatInt(entry.Int(), 10)}

			case reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64:
				vals[name] = []string{strconv.FormatUint(entry.Uint(), 10)}

			case reflect.Slice:
				elem := entry.Type().Elem()
				if elem.Kind() != reflect.String {
					return &FieldTypeError{Field: name, Type: entry.Type()}
				}

				// Copy all of the elements into the new string type
				// The inner string type has been aliased. We need to convert each element
				nEntries := entry.Len()
				set := make([]string, nEntries)
				for i := 0; i < nEntries; i++ {
					set[i] = entry.Index(i).String()
				}

				vals[name] = set

			case reflect.String:
				vals[name] = []string{entry.String()}

			default:
				return &FieldTypeError{Field: name, Type: entry.Type()}
			}
		}

	} // for

	return nil
}

func getFieldEncoder(v reflect.Value) (FieldEncoder, bool) {
	if fieldEncoder, ok := getFieldEncoderOnce(v); ok {
		return fieldEncoder, true
	}

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		// Get the elem
		// Does this even work?
		if fieldEncoder, ok := getFieldEncoderOnce(v.Elem()); ok {
			return fieldEncoder, true
		}
	}

	// Check the pointer in all cases.
	if v.CanAddr() {
		if fieldEncoder, ok := getFieldEncoderOnce(v.Addr()); ok {
			return fieldEncoder, true
		}
	}

	return nil, false
}

func getFieldEncoderOnce(v reflect.Value) (FieldEncoder, bool) {
	if v.CanInterface() {
		if fieldEncoder, ok := v.Interface().(FieldEncoder); ok {
			return fieldEncoder, true
		}
	} // if

	return nil, false
}
