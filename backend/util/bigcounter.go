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

// Adds big.Int number to the internal counting value.
// It modifies the internal state. Only positive integer are allowed.
// Returns false if the value to add is negative and keeps the counter as is.
func (n *BigCounter) AddBigInt(val *big.Int) (*BigCounter, bool) {
	if val.IsUint64() {
		return n.AddUint64(val.Uint64()), true
	}
	// Ignore negative numbers
	if val.Cmp(big.NewInt(0)) == -1 {
		return n, false
	}
	if !n.isExtended() {
		n.initExtended()
	}
	n.extended.Add(n.extended, val)
	return n, true
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
		// The negative value is not supported.
		return nil
	}
	return NewBigCounter(uint64(val))
}

// Constructs a new big counter instance from the big.Int value.
func NewBigCounterFromBigInt(val *big.Int) *BigCounter {
	if val.IsUint64() {
		return NewBigCounter(val.Uint64())
	}

	// The negative value is not supported.
	if val.Cmp(big.NewInt(0)) == -1 {
		return nil
	}

	return &BigCounter{
		base:     math.MaxUint64,
		extended: big.NewInt(0).Set(val),
	}
}
