package form

import (
	"reflect"
	"testing"
)

type recurseTestCase struct {
	key  string
	want []string
}

func TestNestedMapDecodeEncode(t *testing.T) {
	tests := []recurseTestCase{
		{
			"",
			[]string{""},
		},
		{
			"key",
			[]string{"key"},
		},
		{
			"key1[key2]",
			[]string{"key1", "key2"},
		},
		{
			"key1[key2[key3]]",
			[]string{"key1", "key2", "key3"},
		},
		{
			"key1[key2][key3]",
			[]string{"key1[key2][key3]"},
		},
		{
			"key1[key]2[key3]]",
			[]string{"key1[key]2[key3]]"},
		},
	}
	testDecodeEncode(t, tests, NestedMapDecodeFunc, NestedMapEncodeFunc)
}

func TestListMapDecodeEncode(t *testing.T) {
	tests := []recurseTestCase{
		{
			"",
			[]string{""},
		},
		{
			"key",
			[]string{"key"},
		},
		{
			"key1[key2]",
			[]string{"key1", "key2"},
		},
		{
			"key1[key2[key3]]",
			[]string{"key1[key2[key3]]"},
		},
		{
			"key1[key2][key3]",
			[]string{"key1", "key2", "key3"},
		},
		{
			"key1]key2[[key3]",
			[]string{"key1]key2[[key3]"},
		},
		{
			"key1[key2]]key3[",
			[]string{"key1[key2]]key3["},
		},
		{
			"key1[key]2[key3]]",
			[]string{"key1[key]2[key3]]"},
		},
	}
	testDecodeEncode(t, tests, ListMapDecodeFunc, ListMapEncodeFunc)
}

func TestListDecodeEncode(t *testing.T) {
	tests := []recurseTestCase{
		{
			"",
			[]string{""},
		},
		{
			"key",
			[]string{"key"},
		},
		{
			"key1.key2.key3",
			[]string{"key1", "key2", "key3"},
		},
	}
	testDecodeEncode(t, tests, ListDecodeFunc, ListEncodeFunc)
}

func testDecodeEncode(t *testing.T, tests []recurseTestCase, decode DecodeSubKeyFunc, encode EncodeSubKeyFunc) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			keyParts := decode(tt.key)
			if !reflect.DeepEqual(keyParts, tt.want) {
				t.Fatalf("DecodeSubKeyFunc(%q) %#v - want %#v", tt.key, keyParts, tt.want)
			}
			encoded := encode(keyParts)
			if encoded != tt.key {
				t.Fatalf("EncodeSubKeyFunc(%#v) %q - want %q", keyParts, encoded, tt.key)
			}
		})
	}
}
