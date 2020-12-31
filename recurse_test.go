package form

import (
	"reflect"
	"testing"
)

func TestUnwrapKeys(t *testing.T) {
	testCases := []struct {
		key    string
		recuse RecursionFunc
		keys   []string
	}{
		{
			"",
			NestedMapRecurse,
			[]string{""},
		},
		{
			"key",
			NestedMapRecurse,
			[]string{"key"},
		},
		{
			"key1[key2]",
			NestedMapRecurse,
			[]string{"key1", "key2"},
		},
		{
			"key1[key2[key3]]",
			NestedMapRecurse,
			[]string{"key1", "key2", "key3"},
		},
		{
			"key1[key2[]]",
			NestedMapRecurse,
			[]string{"key1", "key2", ""},
		},
		{
			"",
			ListMapRecurse,
			[]string{""},
		},
		{
			"key",
			ListMapRecurse,
			[]string{"key"},
		},
		{
			"key1[key2]",
			ListMapRecurse,
			[]string{"key1", "key2"},
		},
		{
			"key1[key2][key3]",
			ListMapRecurse,
			[]string{"key1", "key2", "key3"},
		},
		{
			"key1[key2][]",
			ListMapRecurse,
			[]string{"key1", "key2", ""},
		},
		{
			"",
			ListRecurse,
			[]string{""},
		},
		{
			"key",
			ListRecurse,
			[]string{"key"},
		},
		{
			"key1.key2",
			ListRecurse,
			[]string{"key1", "key2"},
		},
		{
			"key1.key2.key3",
			ListRecurse,
			[]string{"key1", "key2", "key3"},
		},
		{
			"key1.key2.",
			ListRecurse,
			[]string{"key1", "key2", ""},
		},
	}

	for _, tc := range testCases {
		if keys := expandKey(tc.key, tc.recuse); !reflect.DeepEqual(tc.keys, keys) {
			t.Errorf("Unexpected keys for %q. Expected %#v, found %#v", tc.key, tc.keys, keys)
		}
	}
}
