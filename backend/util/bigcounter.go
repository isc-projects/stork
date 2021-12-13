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
	state kernel
}

// Create new instance of the big counter with the same value.
func (n *BigCounter) clone() *BigCounter {
	kernel := n.state.clone()
	return newBigCounter(kernel)
}

// Add the counting values and return new instance of the big counter.
// Doesn't change the internal state.
func (n *BigCounter) Add(other *BigCounter) *BigCounter {
	number := n.clone()
	number.AddInPlace(other)
	return number
}

// Add the other big counter value to the internal counting value.
// It modifies the internal state.
// You should use this function to avoid too many allocations.
func (n *BigCounter) AddInPlace(other *BigCounter) {
	if other.state.isNaN() {
		// If other is NaN then result is NaN too.
		n.state = newKernelNaN()
	} else if !n.state.canAdd(other.state) {
		// Check if the current counter implementation can store
		// the addition result. If no then switch to big integers.
		n.state = newKernelBigInt(n.state.toBigInt())
	}
	// Add the counting values.
	n.state.addInPlace(other.state)
}

// Add the int64 number to the counting value and return new big counter.
// Doesn't change the internal state.
func (n *BigCounter) AddInt64(val int64) *BigCounter {
	number := NewBigCounter(val)
	return n.Add(number)
}

// Add the int64 number to the internal counting value.
// It modifies the internal state.
func (n *BigCounter) AddInt64InPlace(val int64) {
	number := NewBigCounter(val)
	n.AddInPlace(number)
}

// Divide the this counter by other.
// Doesn't change the internal state.
// If the result is above float64-range then it returns the infinity.
// If the one of the counters is NaN then it returns NaN.
func (n *BigCounter) DivideBy(other *BigCounter) float64 {
	// Check if counters have the same range.
	if n.state.canDivideBy(other.state) {
		// If yes then they can be safety divided.
		return n.state.divideBy(other.state)
	}
	// If no then this counter must be temporarily promoted to a wider range.
	return newKernelBigInt(n.state.toBigInt()).divideBy(other.state)
}

// Works as the Divide function but returns 0 when the value
// of the other counter is 0.
func (n *BigCounter) DivideBySafe(other *BigCounter) float64 {
	if !other.state.isNaN() && other.state.toInt64() == 0 {
		return 0
	}
	return n.DivideBy(other)
}

// Returns the counting value as int64. If the value is above the range
// then returns the maximum value of int64. For NaN returns 0.
func (n *BigCounter) ToInt64() int64 {
	return n.state.toInt64()
}

// Returns the counting value as int64. If the value is above the range
// then returns the @above value. For NaN return @nan.
func (n *BigCounter) ToInt64OrDefault(nan int64, above int64) int64 {
	if n.state.isNaN() {
		return nan
	}
	if !n.state.isIn64BitRange() {
		return above
	}
	return n.state.toInt64()
}

// Returns the counting value as big int. For NaN returns 0.
func (n *BigCounter) ToBigInt() *big.Int {
	return n.state.toBigInt()
}

// Constructs a new big counter instance
// and initialize it with the provided value.
func NewBigCounter(val int64) *BigCounter {
	return newBigCounter(newKernelInt64(val))
}

// Constructs the NaN counter.
// Any operation on it causes NaN or 0.
func NewBigCounterNaN() *BigCounter {
	return newBigCounter(newKernelNaN())
}

// Internal constructor that uses the specific
// counting implementation.
func newBigCounter(k kernel) *BigCounter {
	return &BigCounter{k}
}

// The counting implementation.
type kernel interface {
	// Returns true if the internal value is in range of the int64.
	isIn64BitRange() bool
	// Returns true if the result of the addition can be assigned to it.
	canAdd(k kernel) bool
	// Assigns the addition result to the internal state.
	// It doesn't check the integer overflow.
	addInPlace(k kernel)
	// Returns true if both the counting values have the same range.
	canDivideBy(k kernel) bool
	// Divides the counting values. Returns the infinity if the result
	// is out of float64 range or NaN is any counting value is NaN.
	divideBy(k kernel) float64
	// Returns the counting value as int64. If the value is above int64 range
	// then it returns the maximum int64 value. For NaN returns 0.
	toInt64() int64
	// Indicates that the counting value is NaN.
	isNaN() bool
	// Returns the counting value as big int.
	toBigInt() *big.Int
	// Returns the counting value as float64. Returns an infinity if the counting
	// value is above the float64 range.
	toFloat64() float64
	// Returns new instance with the same counting value.
	clone() kernel
}

// NaN counting value.
// Any operation on it causes NaN or 0.
type kernelNaN struct{}

// Constructs the NaN counting value.
func newKernelNaN() kernel {
	return &kernelNaN{}
}

// Always true.
func (k *kernelNaN) canAdd(other kernel) bool {
	return true
}

// Do nothing.
func (k *kernelNaN) addInPlace(other kernel) {}

// Always true.
func (k *kernelNaN) isNaN() bool {
	return true
}

// Returns 0.
func (k *kernelNaN) toBigInt() *big.Int {
	return big.NewInt(0)
}

// Returns 0.
func (k *kernelNaN) toInt64() int64 {
	return 0
}

// Returns NaN.
func (k *kernelNaN) toFloat64() float64 {
	return math.NaN()
}

// New NaN counting value.
func (k *kernelNaN) clone() kernel {
	return newKernelNaN()
}

// Always true.
func (k *kernelNaN) isIn64BitRange() bool {
	return true
}

// Always true.
func (k *kernelNaN) canDivideBy(other kernel) bool {
	return true
}

// Always NaN.
func (k *kernelNaN) divideBy(other kernel) float64 {
	return math.NaN()
}

// The int64-based counting value.
type kernelInt64 struct {
	value int64
}

// Construct new int64-based counting value.
func newKernelInt64(val int64) kernel {
	return &kernelInt64{value: val}
}

// Return true if result of the addition is in the int64 range.
func (k *kernelInt64) canAdd(other kernel) bool {
	return other.toInt64() <= math.MaxInt64-k.value
}

// Add the counting value to the internal state.
// It may cause integer overflow.
func (k *kernelInt64) addInPlace(other kernel) {
	k.value += other.toInt64()
}

// Always false.
func (k *kernelInt64) isNaN() bool {
	return false
}

// Casts the counting value to big int.
func (k *kernelInt64) toBigInt() *big.Int {
	return big.NewInt(k.value)
}

// Just returns the internal counting value.
func (k *kernelInt64) toInt64() int64 {
	return k.value
}

// Casts the counting value to float64.
func (k *kernelInt64) toFloat64() float64 {
	return float64(k.value)
}

// Returns new instance with the same counting value.
func (k *kernelInt64) clone() kernel {
	return newKernelInt64(k.value)
}

// Always true.
func (k *kernelInt64) isIn64BitRange() bool {
	return true
}

// Return true if another counting value is in the int64 range.
func (k *kernelInt64) canDivideBy(other kernel) bool {
	return other.isIn64BitRange()
}

// Casts the counting value to float64 and divide.
func (k *kernelInt64) divideBy(other kernel) float64 {
	return k.toFloat64() / other.toFloat64()
}

// The big int-based counting value.
type kernelBigInt struct {
	value *big.Int
}

// Constructs the vig int-based counting value.
func newKernelBigInt(val *big.Int) kernel {
	return &kernelBigInt{value: val}
}

// Always true.
func (k *kernelBigInt) canAdd(other kernel) bool {
	return true
}

// Add another counting value to the internal state.
// Casts the another value to big int if necessary.
func (k *kernelBigInt) addInPlace(other kernel) {
	k.value.Add(k.value, other.toBigInt())
}

// Always false.
func (k *kernelBigInt) isNaN() bool {
	return false
}

// Just returns counting value.
func (k *kernelBigInt) toBigInt() *big.Int {
	return k.value
}

// If the counting value is above (or below) int64 range
// then returns maximum (or minimum) int64 value. Otherwise returns
// the counting value.
func (k *kernelBigInt) toInt64() int64 {
	if k.value.IsInt64() {
		return k.value.Int64()
	}
	if k.value.Sign() > 0 {
		return math.MaxInt64
	}
	return math.MinInt64
}

// If the counting value is above (or below) float64 range
// then returns infinity (or minus infinity). Otherwise,
// casts the counting value to float64.
func (k *kernelBigInt) toFloat64() float64 {
	if k.value.IsInt64() {
		return float64(k.value.Int64())
	}
	return math.Inf(k.value.Sign())
}

// Creates new instance with the same counting value.
func (k *kernelBigInt) clone() kernel {
	newValue := new(big.Int)
	newValue.Set(k.value)
	return newKernelBigInt(newValue)
}

// Checks if the counting value is in the int64 range.
func (k *kernelBigInt) isIn64BitRange() bool {
	return k.value.IsInt64()
}

// Always true.
func (k *kernelBigInt) canDivideBy(other kernel) bool {
	return true
}

// If the division result is above (or below) float64 range
// then returns infinity (or minus infinity).
func (k *kernelBigInt) divideBy(other kernel) float64 {
	if other.isNaN() {
		return math.NaN()
	}
	kFLoat := new(big.Float).SetInt(k.toBigInt())
	otherFloat := new(big.Float).SetInt(other.toBigInt())
	div := new(big.Float).Quo(kFLoat, otherFloat)
	res, _ := div.Float64()
	return res
}
