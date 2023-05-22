package storkutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hash/fnv"
)

// Convenience function creating FNV128 hash from input string.
func Fnv128(input string) string {
	h := fnv.New128()
	// Ignore errors because they are never returned in this case.
	_, _ = h.Write([]byte(input))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// Convenience function creating FNV128 hash from any value. It prints
// the value in the default format and hashes using the Fnv128 function.
func Fnv128AnyValue(value any) string {
	return Fnv128(fmt.Sprintf("%+v", value))
}

// Convenience function generating random bytes of the specified
// length and encoding them with base64.
func Base64Random(length int) (hash string, err error) {
	b := make([]byte, length)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	hash = base64.StdEncoding.EncodeToString(b)
	return
}
