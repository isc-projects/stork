package storkutil

import (
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
