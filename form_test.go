package form

import (
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	testStruct := struct {
		a string
		B string
		C string `form:"D"`
	}{}

	vals := url.Values{
		"B": []string{"B value"},
		"C": []string{"C value"},
		"D": []string{"D value"},
	}
	if err := Parse(vals, &testStruct); err != nil {
		t.Errorf("Parse: %q", err)
	} // if
}
