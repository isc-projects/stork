package storkutil

import (
	"math/big"

	"github.com/pkg/errors"
)

// A wrapper for big.Int that can be marshaled and unmarshaled to JSON. It is
// intended to handle big numbers from Kea API without losing precision.
// It doesn't prepend the `n` literal as it isn't supported by the Kea parser
// and JSON standard.
//
// Warning: This type SHOULD NOT be used to serialize big numbers returned by
// the Stork API. If the serialized value by this method is processed by the
// JSON-compliant parser, it is cast to a float64, so the precision is lost.
//
// See: https://stackoverflow.com/a/53991836
type BigIntJSON struct {
	value big.Int
}

func NewBigIntJSONFromInt64(value int64) BigIntJSON {
	return BigIntJSON{*big.NewInt(value)}
}

// Return the big int instance.
func (b BigIntJSON) BigInt() *big.Int {
	return &b.value
}

// Serializes the big integer to JSON as a numeric literal.
func (b BigIntJSON) MarshalJSON() ([]byte, error) {
	return []byte(b.value.String()), nil
}

// Deserializes the big integer from JSON numeric literal without casting it
// intermediate to float64.
func (b *BigIntJSON) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	var z big.Int
	_, ok := z.SetString(string(p), 10)
	if !ok {
		return errors.Errorf("not a valid big integer: %s", p)
	}
	b.value = z
	return nil
}
