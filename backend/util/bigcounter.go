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

// Indicates that the uint64 can be added to the base value without integer overflow.
func (n *BigCounter) canAddToBase(val uint64) bool {
	return n.base <= math.MaxUint64-val
}

// Indicates that another counting value can be added to the internal state.
func (n *BigCounter) canAdd(other *BigCounter) bool {
	if n.isExtended() {
		return true
	}
	if other.isExtended() {
		return false
	}
	return n.canAddToBase(other.base)
}

// Indicates that this counter uses big-int based counter.
func (n *BigCounter) isExtended() bool {
	return n.extended != nil
}

// Initializes the big-int counter with current value of the uint64 counter.
// Set the uint64 counter to max uint64 value to ensure that the canAddToBase will return false.
// It should be called only once per big counter.
// It should be called only if the counting value exceeds the uint64 range.
func (n *BigCounter) initExtended() {
	n.extended = big.NewInt(0).SetUint64(n.base)
	n.base = math.MaxUint64
}

// Adds the other big counter value to the internal counting value.
// It modifies the internal state.
func (n *BigCounter) Add(other *BigCounter) *BigCounter {
	if !n.canAdd(other) {
		n.initExtended()
	}

	if n.isExtended() {
		n.extended.Add(n.extended, other.ToBigInt())
	} else {
		n.base += other.base
	}

	return n
}

// Adds uint64 number to the internal counting value.
// It modifies the internal state.
func (n *BigCounter) AddUint64(val uint64) *BigCounter {
	if !n.isExtended() && !n.canAddToBase(val) {
		n.initExtended()
	}

	if n.isExtended() {
		valBig := new(big.Int).SetUint64(val)
		n.extended.Add(n.extended, valBig)
	} else {
		n.base += val
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

// Works as the Divide function but returns 0 when the value
// of the denominator counter is 0.
func (n *BigCounter) DivideSafeBy(other *BigCounter) float64 {
	if !other.isExtended() && other.base == 0 {
		return 0.0
	}
	return n.DivideBy(other)
}

// Returns the counting value as int64. If the value is above the range
// then returns the maximum value of int64.
func (n *BigCounter) ToInt64() int64 {
	if n.base >= math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(n.base)
}

// Returns the counting value as uint64. If the value is above the range
// then returns the maximum value of uint64. If the value is below the range
// then returns 0.
func (n *BigCounter) ToUint64() uint64 {
	if n.isExtended() {
		if n.extended.IsUint64() {
			return n.extended.Uint64()
		}
		return math.MaxUint64
	}

	return n.base
}

// Returns the counting value as big int.
func (n *BigCounter) ToBigInt() *big.Int {
	if n.isExtended() {
		return n.extended
	}
	return big.NewInt(0).SetUint64(n.base)
}

// Constructs a new big counter instance
// and initializes it with the provided value.
func NewBigCounter(val uint64) *BigCounter {
	return &BigCounter{
		base:     val,
		extended: nil,
	}
}
