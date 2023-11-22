package storkutil

// An interface to a hasher generating hash value from an
// intput string. Implementations may use different algorithms
// for generating the hashes.
type Hasher interface {
	Hash(input any) string
}
