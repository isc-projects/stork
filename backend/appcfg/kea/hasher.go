package keaconfig

import storkutil "isc.org/stork/util"

// A constant that can be modified to influence the change of the
// hasher output. The Hasher produces an output depending on the
// hashed input value and the hasher sequence. If the sequence is
// stable, the hasher produces the same output for a given input
// value. Sometimes, however, it is required to influence the change
// of the hashes to force Stork fetch updated Kea configurations.
// It is often the case after applying bug fixes that require fetching
// configurations to take effect. In such cases, a developer willing
// to force configuration fetch, should bump this value causing the
// hashes change.
const hasherSequence int64 = 1

var _ storkutil.Hasher = (*Hasher)(nil)

// A hasher used for Kea configuration hashing.
type Hasher struct {
	// Hasher sequence.
	seq int64
}

// Creates new hasher instance using the hasherSequence constant.
func NewHasher() *Hasher {
	return &Hasher{
		seq: hasherSequence,
	}
}

// Hashes the input values and returns the hash. It uses the Fnv128
// hashing technique.
func (h Hasher) Hash(input any) string {
	return storkutil.Fnv128(h.seq, input)
}
