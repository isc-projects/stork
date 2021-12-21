package storkutil

import (
	"math"
	"math/big"
)

// The utility to count the large items - e.g. IPv6 addresses.
// It scales internally with the counting value - starts from
// int64-based counter and switches to bigInt only if necessary.
// Additionally, it supports the Not-a-Number state and calculates
// the proportion between two counters.
type BigCounter struct {
	// Counting value wrapper.
	base     int64
	extended *big.Int
}

// Indicates that the int64 can be added to the base value without integer overflow.
func (n *BigCounter) canAddToBase(val int64) bool {
	return val < math.MaxInt64-n.base
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

// Indicates that this counter uses int64 based counter or big-int based counter.
func (n *BigCounter) isExtended() bool {
	return n.extended != nil
}

// Initializes the big-int counter with current value of the int64 counter.
// Set the int64 counter to max int64 value to ensure that the canAddToBase will return false.
// It should be called only once per big counter.
// It should be called only if the counting value exceeds the int64 range.
func (n *BigCounter) initExtended() {
	n.extended = big.NewInt(n.base)
	n.base = math.MaxInt64
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

// Adds the int64 number to the internal counting value.
// It modifies the internal state.
func (n *BigCounter) AddInt64(val int64) *BigCounter {
	if !n.isExtended() && !n.canAddToBase(val) {
		n.initExtended()
	}

	if n.isExtended() {
		n.extended.Add(n.extended, big.NewInt(val))
	} else {
		n.base += val
	}

	return n
}

// Adds uint64 number to the internal counting value.
// It modifies the internal state.
func (n *BigCounter) AddUInt64(val uint64) *BigCounter {
	if !n.isExtended() {
		if val <= math.MaxInt64 && n.canAddToBase(int64(val)) {
			n.base += int64(val)
			return n
		}
		n.initExtended()
	}
	valBig := new(big.Int).SetUint64(val)
	n.extended.Add(n.extended, valBig)
	return n
}

// Divides this counter by the other.
// Doesn't change the internal state.
// If the result is above float64-range then it returns the infinity.
func (n *BigCounter) DivideBy(other *BigCounter) float64 {
	if !n.isExtended() && !other.isExtended() {
		return float64(n.base) / float64(other.base)
	}

	nFLoat := new(big.Float).SetInt(n.ToBigInt())
	otherFloat := new(big.Float).SetInt(other.ToBigInt())
	div := new(big.Float).Quo(nFLoat, otherFloat)
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
	return n.base
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

	if n.base >= 0 {
		return uint64(n.base)
	}

	return 0
}

// Returns the counting value as big int.
func (n *BigCounter) ToBigInt() *big.Int {
	if n.isExtended() {
		return n.extended
	}
	return big.NewInt(n.base)
}

// Constructs a new big counter instance
// and initializes it with the provided value.
func NewBigCounter(val int64) *BigCounter {
	return &BigCounter{
		base:     val,
		extended: nil,
	}
}
