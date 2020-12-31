package form

import (
	"strings"
)

// RecursionFunc is used by the parser to unwrap the keys and values supplied by the form into
// sub objects
type RecursionFunc func(key string) (baseKey string, subKey string, ok bool)

func expandKey(key string, recurse RecursionFunc) []string {
	var keys []string
	for {
		if nextKey, subKey, ok := recurse(key); ok {
			keys = append(keys, nextKey)
			key = subKey
		} else {
			keys = append(keys, key)
			break
		}
	}

	return keys
}

// NestedMapRecurse unwraps keys of the type key1[key2[key3]].
func NestedMapRecurse(key string) (baseKey string, subKey string, ok bool) {
	if key == "" || key[len(key)-1] != ']' {
		return "", "", false
	}

	idx := strings.Index(key, "[")
	if idx == -1 {
		return "", "", false
	}

	baseKey = key[:idx]
	subKey = key[idx+1 : len(key)-1]
	ok = true

	return baseKey, subKey, ok
}

// ListMapRecurse unwraps forms of the type key1[key2][key3]
func ListMapRecurse(key string) (baseKey string, subKey string, ok bool) {
	if key == "" {
		return "", "", false
	}

	openIdx := strings.Index(key, "[")
	if openIdx == -1 {
		return "", "", false
	}
	closeIdx := strings.Index(key, "]")
	if closeIdx == -1 || openIdx > closeIdx {
		return "", "", false
	}

	baseKey = key[:openIdx]
	subKey = key[openIdx+1:closeIdx] + key[closeIdx+1:]
	ok = true

	return baseKey, subKey, ok
}

// ListRecurse unwraps forms of the type key1.key2.key3
func ListRecurse(key string) (baseKey string, subKey string, ok bool) {
	if key == "" {
		return "", "", false
	}

	baseKey, subKey = firstSplit(key, ".")
	if baseKey == "" || baseKey == key {
		return "", "", false
	}
	ok = true

	return baseKey, subKey, ok
}

func firstSplit(s, split string) (first, second string) {
	idx := strings.Index(s, split)
	if idx == -1 {
		return s, ""
	}

	return s[:idx], s[idx+1:]
}
