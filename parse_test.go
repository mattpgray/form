package form

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func TestParseString(t *testing.T) {
	type Alias string
	testStruct := struct {
		A string
		B Alias
	}{}

	vals := url.Values{
		"A": []string{"A value"},
		"B": []string{"B value"},
	}
	if err := NewDecoder().Decode(vals, &testStruct); err != nil {
		t.Fatalf("NewDecoder().Decode: %q", err)
	} // if

	if testStruct.A != "A value" {
		t.Errorf("Unexpected value in A. Expected \"A value\" found %q", testStruct.A)
	}
	if testStruct.B != "B value" {
		t.Errorf("Unexpected value in B. Expected \"B value\" found %q", testStruct.B)
	}

	vals = url.Values{
		"A": []string{"A value", "A value 2"},
	}

	var fieldParseErr *FieldParseError
	var unexpectedValuesError *UnexpectedValuesError
	if err := NewDecoder().Decode(vals, &testStruct); !errors.As(err, &fieldParseErr) || !errors.As(err, &unexpectedValuesError) {
		t.Fatalf("NewDecoder().Decode not nil: %q", err)
	} else {
		if fieldParseErr.Field != "A" {
			t.Errorf("Unexpected field in error. Expected \"A\" found %q", fieldParseErr.Field)
		}
	}
}

func TestParseStringSlice(t *testing.T) {
	type GlobalAlias []string
	type InnerAlias string
	testStruct := struct {
		A []string
		B GlobalAlias
		C []InnerAlias
	}{}

	as := []string{"A value 1", "A Value 2"}
	bs := []string{"B value 1", "B Value 2"}
	cs := []string{"C value 1", "C Value 2"}

	vals := url.Values{
		"A": as,
		"B": bs,
		"C": cs,
	}
	if err := NewDecoder().Decode(vals, &testStruct); err != nil {
		t.Fatalf("NewDecoder().Decode: %q", err)
	} // if

	if !reflect.DeepEqual(testStruct.A, as) {
		t.Errorf("Unexpected value in A. Expected %v found %v", as, testStruct.A)
	}
	if !reflect.DeepEqual([]string(testStruct.B), bs) {
		t.Errorf("Unexpected value in B. Expected %v found %v", bs, testStruct.B)
	}

	var cRes []string
	if testStruct.C != nil {
		cRes = make([]string, len(testStruct.C))

		for i := range testStruct.C {
			cRes[i] = string(testStruct.C[i])
		}
	}
	if !reflect.DeepEqual(cRes, cs) {
		t.Errorf("Unexpected value in B. Expected %v found %v", cs, testStruct.C)
	}
}

func TestParseInt(t *testing.T) {
	testStruct := struct {
		Int   int
		Int8  int8
		Int16 int16
		Int32 int32
		Int64 int64
	}{}

	vals := url.Values{
		"Int":   []string{"1"},
		"Int8":  []string{"2"},
		"Int16": []string{"3"},
		"Int32": []string{"4"},
		"Int64": []string{"5"},
	}
	if err := NewDecoder().Decode(vals, &testStruct); err != nil {
		t.Fatalf("NewDecoder().Decode: %q", err)
	}

	const (
		intVal   = 1
		int8Val  = 2
		int16Val = 3
		int32Val = 4
		int64Val = 5
	)

	if testStruct.Int != intVal {
		t.Errorf("Unexpected value in Int. Expected %d, found %d", intVal, testStruct.Int)
	}

	if testStruct.Int8 != int8Val {
		t.Errorf("Unexpected value in Int8. Expected %d, found %d", int8Val, testStruct.Int8)
	}

	if testStruct.Int16 != int16Val {
		t.Errorf("Unexpected value in Int16. Expected %d, found %d", int16Val, testStruct.Int16)
	}

	if testStruct.Int32 != int32Val {
		t.Errorf("Unexpected value in Int32. Expected %d, found %d", int32Val, testStruct.Int32)
	}

	if testStruct.Int64 != int64Val {
		t.Errorf("Unexpected value in Int64. Expected %d, found %d", int64Val, testStruct.Int64)
	}

	vals = url.Values{
		"Int": []string{"notanint"},
	}
	var fieldParseErr *FieldParseError
	var unexpectedValueError *UnexpectedValueError
	if err := NewDecoder().Decode(vals, &testStruct); !errors.As(err, &fieldParseErr) || !errors.As(err, &unexpectedValueError) {
		t.Fatalf("NewDecoder().Decode not nil: %q", err)
	} else {
		if fieldParseErr.Field != "Int" {
			t.Errorf("Unexpected field in fieldParseErr. Expected \"Int\" found %q", fieldParseErr.Field)
		}

		if unexpectedValueError.Value != "notanint" {
			t.Errorf("Unexpected value in unexpectedValueError. Expected \"notanint\" found %q", unexpectedValueError.Value)
		}
	}
}

func TestParseRecursive(t *testing.T) {
	type InnerStruct struct {
		Value string
	}

	type OuterStruct struct {
		Value string
		Inner InnerStruct
	}

	var outer OuterStruct

	const outerVal = "outer"
	const innerVal = "inner"
	vals := url.Values{
		"Value":        []string{outerVal},
		"Inner[Value]": []string{innerVal},
	}

	d := NewDecoder()
	d.Recurse(NestedMapDecodeFunc)
	if err := d.Decode(vals, &outer); err != nil {
		t.Fatalf("NewDecoder().Decode: %q", err)
	}

	if outer.Value != outerVal {
		t.Errorf("Unexpected value in outer.Value. Expected %q, found %q", outerVal, outer.Value)
	}
	if outer.Inner.Value != innerVal {
		t.Errorf("Unexpected value in outer.Inner.Value. Expected %q, found %q", innerVal, outer.Inner.Value)
	}
}

func TestParseAnonymous(t *testing.T) {
	type InnerStruct struct {
		InnerValue string
	}

	type OuterStruct struct {
		InnerStruct
		Value string
	}

	var outer OuterStruct

	const outerVal = "outer"
	const innerVal = "inner"
	vals := url.Values{
		"Value":      []string{outerVal},
		"InnerValue": []string{innerVal},
	}

	if err := NewDecoder().Decode(vals, &outer); err != nil {
		t.Fatalf("NewDecoder().Decode: %q", err)
	}

	if outer.Value != outerVal {
		t.Errorf("Unexpected value in outer.Value. Expected %q, found %q", outerVal, outer.Value)
	}
	if outer.InnerValue != innerVal {
		t.Errorf("Unexpected value in outer.InnerValue. Expected %q, found %q", innerVal, outer.InnerValue)
	}
}
