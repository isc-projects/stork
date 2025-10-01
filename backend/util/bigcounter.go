package storkutil

import (
	"math"
	"math/big"
)

// The utility to count the large items - e.g. IPv6 addresses.
// It scales internally with the counting value - starts from
// uint64-based counter and switches to bigInt only if necessary.
// It has better performance than raw big int objects in the uint64
// range (9-10x faster than big int, 10-12x slower than raw uint64) and almost
// the same performance outside this range (7% slower). It improves the
// overall performance because in most cases (IPv4 and small IPv6 networks)
// we won't need big numbers.
// Additionally, it calculates the proportion between two counters.
type BigCounter struct {
	// Counting value wrapper.
	base     uint64
	extended *big.Int
}

// Indicates that two uint64 would cause integer overflow if added together.
func addWouldOverflow(a, b uint64) bool {
	return a > math.MaxUint64-b
}

// Indicates that subtracting `b` from `a` would cause integer underflow.
func subWouldUnderflow(a, b uint64) bool {
	return a < b
}

// Indicates that this counter uses big-int based counter.
func (n *BigCounter) isExtended() bool {
	return n.extended != nil
}

// Normalizes the big counter to the uint64 range if possible.
func (n *BigCounter) normalize() {
	if n.isExtended() && n.extended.IsUint64() {
		n.base = n.extended.Uint64()
		n.extended = nil
	}
}

// Adds two big counters and puts the result into the receiver.
func (n *BigCounter) Add(a, b *BigCounter) *BigCounter {
	if a.extended != nil || b.extended != nil || addWouldOverflow(a.base, b.base) {
		outBigInt := n.extended
		if outBigInt == nil {
			outBigInt = big.NewInt(0)
		}

		n.extended = outBigInt.Add(a.ToBigInt(), b.ToBigInt())
		n.base = math.MaxUint64
	} else {
		n.base = a.base + b.base
		n.extended = nil
	}

	return n
}

// Subtracts the big counters and puts the result into the receiver.
func (n *BigCounter) Subtract(a, b *BigCounter) *BigCounter {
	if a.isExtended() || b.isExtended() || subWouldUnderflow(a.base, b.base) {
		outBigInt := n.extended
		if outBigInt == nil {
			outBigInt = big.NewInt(0)
		}

		n.extended = outBigInt.Sub(n.ToBigInt(), b.ToBigInt())
		n.base = 0
		n.normalize()
	} else {
		n.base = a.base - b.base
		n.extended = nil
	}
	return n
}

// Adds the big counter and the uint64 value and puts the result into the
// receiver.
func (n *BigCounter) AddUint64(a *BigCounter, b uint64) *BigCounter {
	if a.isExtended() || addWouldOverflow(a.base, b) {
		outBigInt := n.extended
		if outBigInt == nil {
			outBigInt = big.NewInt(0)
		}

		n.extended = outBigInt.Add(a.ToBigInt(), big.NewInt(0).SetUint64(b))
		n.base = math.MaxUint64
	} else {
		n.base = a.base + b
		n.extended = nil
	}
	return n
}

// Adds big counter and the big int value and puts the result into the
// receiver.
func (n *BigCounter) AddBigInt(a *BigCounter, b *big.Int) *BigCounter {
	if !a.isExtended() && b.IsUint64() && !addWouldOverflow(a.base, b.Uint64()) {
		n.extended = nil
		n.base = a.base + b.Uint64()
	} else {
		outBigInt := n.extended
		if outBigInt == nil {
			outBigInt = big.NewInt(0)
		}

		n.extended = outBigInt.Add(a.ToBigInt(), b)
		n.base = math.MaxUint64
	}
	return n
}

// Divides this counter by the other.
// Doesn't change the internal state.
// If the result is above float64-range then it returns the infinity.
func (n *BigCounter) DivideBy(other *BigCounter) float64 {
	if !n.isExtended() && !other.isExtended() {
		return float64(n.base) / float64(other.base)
	}

	nFloat := new(big.Float).SetInt(n.ToBigInt())
	otherFloat := new(big.Float).SetInt(other.ToBigInt())
	div := new(big.Float).Quo(nFloat, otherFloat)
	res, _ := div.Float64()
	return res
}

// Indicates if the big counter is zero.
func (n *BigCounter) IsZero() bool {
	if n.isExtended() {
		return n.extended.Sign() == 0
	}
	return n.base == 0
}

// Works as the Divide function but returns 0 when the value
// of the denominator counter is 0.
func (n *BigCounter) DivideSafeBy(other *BigCounter) float64 {
	if other.IsZero() {
		return 0
	}
	return n.DivideBy(other)
}

// Returns the counting value as int64. If the value is above the range
// then returns the maximum value of int64.
func (n *BigCounter) ToInt64() int64 {
	if n.isExtended() {
		if n.extended.IsInt64() {
			return n.extended.Int64()
		}
		if n.extended.Sign() == -1 {
			return math.MinInt64
		}
		return math.MaxInt64
	} else {
		if n.base > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(n.base)
	}
}

// Returns the counting value as float64. If the value exceeds the maximum
// safe integer value (2^53-1) then the precision is lost.
func (n *BigCounter) ToFloat64() float64 {
	if n.isExtended() {
		value, _ := n.extended.Float64()
		return value
	}
	return float64(n.base)
}

// Returns the counting value as uint64. If the value is in range, returns it
// and true. If the value is above the range then returns the maximum value of
// uint64 and false. If the value is below the range then returns 0 and false.
func (n *BigCounter) ToUint64() (uint64, bool) {
	if n.isExtended() {
		if n.extended.IsUint64() {
			return n.extended.Uint64(), true
		}
		// Is below the range.
		if n.extended.Cmp(big.NewInt(0)) == -1 {
			return 0, false
		}
		return math.MaxUint64, false
	}

	return n.base, true
}

// Returns the counting value as big int.
func (n *BigCounter) ToBigInt() *big.Int {
	if n.isExtended() {
		return n.extended
	}
	return big.NewInt(0).SetUint64(n.base)
}

// Returns the counting value as uint64 if the value is in uint64 range.
// Otherwise, returns big.Int.
func (n *BigCounter) ConvertToNativeType() interface{} {
	if n.isExtended() {
		return n.extended
	}
	return n.base
}

// Constructs a new big counter instance
// and initializes it with the provided value.
func NewBigCounter(val uint64) *BigCounter {
	return &BigCounter{
		base:     val,
		extended: nil,
	}
}

// Constructs a new big counter instance from the int64 value.
func NewBigCounterFromInt64(val int64) *BigCounter {
	if val < 0 {
		return &BigCounter{
			base:     0,
			extended: big.NewInt(val),
		}
	}
	return NewBigCounter(uint64(val))
}

// Constructs a new big counter instance from the big.Int value.
func NewBigCounterFromBigInt(val *big.Int) *BigCounter {
	if val.IsUint64() {
		return NewBigCounter(val.Uint64())
	}

	return &BigCounter{
		base:     math.MaxUint64,
		extended: big.NewInt(0).Set(val),
	}
}
