package form

import (
	"log"
	"testing"
)

func TestEncode(t *testing.T) {
	type Alias string
	testStruct := struct {
		A string
		B Alias
	}{}

	s, err := EncodeString(testStruct)
	if err != nil {
		t.Fatalf("EncodeString: %q", err)
	} // if

	log.Printf("%s", s)
}
