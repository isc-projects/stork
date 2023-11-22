package storkutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hash/fnv"
)

// Convenience function creating FNV128 hash from an input value.
func Fnv128(input ...any) string {
	h := fnv.New128()
	// Ignore errors because they are never returned in this case.
	for _, i := range input {
		_, _ = h.Write([]byte(fmt.Sprintf("%+v", i)))
	}
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
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
