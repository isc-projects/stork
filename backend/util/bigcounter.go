package storkutil

import (
	"fmt"
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

// Indicate that the int64 can be added to the base without integer overflow.
func (n *BigCounter) canAddToBase(val int64) bool {
	return val < math.MaxInt64-n.base
}

// Indicate that the another counting value can be added to the internal state.
func (n *BigCounter) canAdd(other *BigCounter) bool {
	if n.isExtended() {
		return true
	}
	if other.isExtended() {
		return false
	}
	return n.canAddToBase(other.base)
}

// Indicate that this counter used int64 based counter or big-int based counter.
func (n *BigCounter) isExtended() bool {
	return n.extended != nil
}

// Initialize the big-int counter with current value of the int64 counter.
// Set the int64 counter to max int64 value to ensure that the canAddToBase will return false.
// It should be called only once per big counter.
// It should be called only if the counting value exceeds the int64 range.
func (n *BigCounter) initExtended() {
	n.extended = big.NewInt(n.base)
	n.base = math.MaxInt64
}

// Add the other big counter value to the internal counting value.
// It modifies the internal state.
// You should use this function to avoid too many allocations.
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

// Add the int64 number to the internal counting value.
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

// Add uint64 number to the internal counting value.
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

// Divide the this counter by other.
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
// of the other counter is 0.
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

// Returns the counting value as big int.
func (n *BigCounter) ToBigInt() *big.Int {
	if n.isExtended() {
		return n.extended
	}
	return big.NewInt(n.base)
}

// Returns the string representation of the counting value.
func (n *BigCounter) String() string {
	if n.isExtended() {
		return fmt.Sprint(n.extended)
	}
	return fmt.Sprint(n.base)
}

// Constructs a new big counter instance
// and initialize it with the provided value.
func NewBigCounter(val int64) *BigCounter {
	return &BigCounter{
		base:     val,
		extended: nil,
	}
}
