package form

import (
	"strings"
)

// DecodeSubKeyFunc is used by the Decoder to unwrap the keys and values supplied by the form into
// sub objects. The slice retunred from these functions must contain at least one element. If the supplied
// key is in the incorrect format then these functions return a slice with one element only containg the
// incorrect key. These functions should not alter the case of the keys in order to achieve case insensitivity.
// Instead, call (*Decoder).StrictCase(false).
type DecodeSubKeyFunc func(key string) (keyParts []string)

// NestedMapDecodeFunc unwraps keys of the type key1[key2[key3]].
func NestedMapDecodeFunc(key string) (keyParts []string) {
	openIdxs := indexAll(key, "[")
	if len(openIdxs) == 0 {
		return singleKey(key)
	}
	// All the close brackets must be at the end
	wantSuffix := strings.Repeat("]", len(openIdxs))
	if !strings.HasSuffix(key, wantSuffix) {
		return singleKey(key)
	}
	strippedKey := key[:len(key)-len(wantSuffix)]
	if strings.Contains(strippedKey, "]") {
		return singleKey(key)
	}
	// Now get the keys
	keyParts = make([]string, 0, len(openIdxs)+1)
	keyParts = append(keyParts, strippedKey[:openIdxs[0]])
	for i := 0; i < len(openIdxs); i++ {
		var keyPart string
		if i == len(openIdxs)-1 {
			keyPart = strippedKey[openIdxs[i]+1:]
		} else {
			keyPart = strippedKey[openIdxs[i]+1 : openIdxs[i+1]]
		}
		keyParts = append(keyParts, keyPart)
	}
	return keyParts
}

// ListMapDecodeFunc unwraps forms of the type key1[key2][key3]
func ListMapDecodeFunc(key string) (keyParts []string) {
	openIdxs := indexAll(key, "[")
	closeIdxs := indexAll(key, "]")
	if len(openIdxs) == 0 || len(openIdxs) != len(closeIdxs) {
		return singleKey(key)
	}
	for i := 0; i < len(openIdxs); i++ {
		if openIdxs[i] > closeIdxs[i] {
			return singleKey(key)
		}
	}
	for i := 0; i < len(openIdxs)-1; i++ {
		if openIdxs[i+1] < closeIdxs[i] {
			return singleKey(key)
		}
	}
	keyParts = make([]string, 0, len(openIdxs)+1)
	keyParts = append(keyParts, key[:openIdxs[0]])
	for i := 0; i < len(openIdxs); i++ {
		keyParts = append(keyParts, key[openIdxs[i]+1:closeIdxs[i]])
	}
	return keyParts
}

// ListDecodeFunc unwraps forms of the type key1.key2.key3
func ListDecodeFunc(key string) (keyParts []string) {
	return strings.Split(key, ".")
}

// EncodeSubKeyFunc is used by the Encoder to join sub keys together into one key.
// The slice passed into the encode function always has at least one element.
type EncodeSubKeyFunc func(keyParts []string) string

// NestedMapEncodeFunc makes keys of the type key1[key2[key3]].
func NestedMapEncodeFunc(keyParts []string) string {
	sb := &strings.Builder{}
	sb.WriteString(keyParts[0])
	for i := 1; i < len(keyParts); i++ {
		sb.WriteString("[")
		sb.WriteString(keyParts[i])
	}
	for i := 1; i < len(keyParts); i++ {
		sb.WriteString("]")
	}
	return sb.String()
}

// ListMapEncodeFunc makes keys of the type key1[key2][key3].
func ListMapEncodeFunc(keyParts []string) string {
	sb := &strings.Builder{}
	sb.WriteString(keyParts[0])
	for i := 1; i < len(keyParts); i++ {
		sb.WriteString("[")
		sb.WriteString(keyParts[i])
		sb.WriteString("]")
	}
	return sb.String()
}

// ListEncodeFunc makes keys of the type key1.key2.key3.
func ListEncodeFunc(keyParts []string) string {
	return strings.Join(keyParts, ".")
}

// indexAll finds the indexes of all non-overlapping instances of the subtring
func indexAll(s string, substr string) []int {
	var idxs []int
	start := 0
	for {
		if idx := strings.Index(s[start:], substr); idx < 0 {
			return idxs
		} else {
			idxs = append(idxs, idx+start)
			start += idx + len(substr)
		}
	}
}

func singleKey(key string) []string {
	return []string{key}
}
