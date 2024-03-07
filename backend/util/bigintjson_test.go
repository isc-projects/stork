package storkutil

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the big integer is marshaled to JSON as a numeric literal.
func TestBigIntJSONMarshal(t *testing.T) {
	// Arrange
	bigInts := []*big.Int{
		// Small integers.
		big.NewInt(0),
		big.NewInt(42),
		big.NewInt(-1),
		// Max standard integer value.
		big.NewInt(0).SetUint64(math.MaxUint64),
		// Exceed the uint64 range.
		big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(1)),
		// Really big number.
		big.NewInt(0).Add(
			big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).SetUint64(math.MaxUint64),
			),
			big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).SetUint64(math.MaxUint64),
			),
		),
	}

	strings := []string{
		"0",
		"42",
		"-1",
		"18446744073709551615",
		"18446744073709551616",
		"73786976294838206460",
	}

	for i := range bigInts {
		t.Run(strings[i], func(t *testing.T) {
			bigInt := BigIntJSON{*bigInts[i]}

			// Act
			bytes, err := json.Marshal(bigInt)

			// Assert
			require.NoError(t, err)
			require.Equal(t, strings[i], string(bytes))
		})
	}
}

// Test that the big integer is unmarshaled from JSON numeric literal without
// casting it intermediate to float64 and losing precision.
func TestBigIntJSONUnmarshal(t *testing.T) {
	// Arrange
	bigInts := []*big.Int{
		// Small integers.
		big.NewInt(0),
		big.NewInt(42),
		big.NewInt(-1),
		// Max standard integer value.
		big.NewInt(0).SetUint64(math.MaxUint64),
		// Exceed the uint64 range.
		big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(1)),
		// Really big number.
		big.NewInt(0).Add(
			big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).SetUint64(math.MaxUint64),
			),
			big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).SetUint64(math.MaxUint64),
			),
		),
	}

	strings := []string{
		"0",
		"42",
		"-1",
		"18446744073709551615",
		"18446744073709551616",
		"73786976294838206460",
	}

	for i := range bigInts {
		t.Run(strings[i], func(t *testing.T) {
			bytes := []byte(strings[i])

			// Act
			var bigInt BigIntJSON
			err := json.Unmarshal(bytes, &bigInt)

			// Assert
			require.NoError(t, err)
			require.Equal(t, bigInts[i], bigInt.BigInt())
		})
	}
}
