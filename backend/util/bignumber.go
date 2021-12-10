package storkutil

import (
	"math"
	"math/big"
)

type BigNumber struct {
	state kernel
}

func (n *BigNumber) clone() *BigNumber {
	kernel := n.state.clone()
	return newBigNumber(kernel)
}

func (n *BigNumber) Add(other *BigNumber) *BigNumber {
	number := n.clone()
	number.AddInPlace(other)
	return number
}

func (n *BigNumber) AddInPlace(other *BigNumber) {
	if !n.state.canAdd(other.state) {
		n.state = newKernelBigInt(n.state.toBigInt())
	}
	n.state.addInPlace(other.state)
}

func (n *BigNumber) AddInt64(val int64) *BigNumber {
	number := NewBigNumber(val)
	return n.Add(number)
}

func (n *BigNumber) AddInt64InPlace(val int64) {
	number := NewBigNumber(val)
	n.AddInPlace(number)
}

func (n *BigNumber) Divide(other *BigNumber) float64 {
	if n.state.canDivide(other.state) {
		return n.state.divide(other.state)
	}

	return newKernelBigInt(n.state.toBigInt()).divide(other.state)
}

func (n *BigNumber) DivideSafe(other *BigNumber) float64 {
	if !other.state.isNaN() && other.state.toInt64() == 0 {
		return 0
	}
	return n.Divide(other)
}

func (n *BigNumber) ToInt64() int64 {
	return n.state.toInt64()
}

func (n *BigNumber) ToInt64OrDefault(nan int64, above int64) int64 {
	if n.state.isNaN() {
		return nan
	}
	if !n.state.isIn64BitRange() {
		return above
	}
	return n.state.toInt64()
}

func NewBigNumber(val int64) *BigNumber {
	return &BigNumber{newKernelInt64(val)}
}

func newBigNumber(k kernel) *BigNumber {
	return &BigNumber{k}
}

func NewBigNumberNaN() *BigNumber {
	return &BigNumber{newKernelNaN()}
}

type kernel interface {
	isIn64BitRange() bool
	canAdd(k kernel) bool
	addInPlace(k kernel)
	canDivide(k kernel) bool
	divide(k kernel) float64
	toInt64() int64
	isNaN() bool
	toBigInt() *big.Int
	toFloat64() float64
	clone() kernel
}

type kernelNaN struct{}

func newKernelNaN() kernel {
	return &kernelNaN{}
}

func (k *kernelNaN) canAdd(other kernel) bool {
	return false
}

func (k *kernelNaN) addInPlace(other kernel) {}

func (k *kernelNaN) isNaN() bool {
	return true
}

func (k *kernelNaN) toBigInt() *big.Int {
	return big.NewInt(0)
}

func (k *kernelNaN) toInt64() int64 {
	return 0
}

func (k *kernelNaN) toFloat64() float64 {
	return math.NaN()
}

func (k *kernelNaN) clone() kernel {
	return newKernelNaN()
}

func (k *kernelNaN) isIn64BitRange() bool {
	return true
}

func (k *kernelNaN) canDivide(other kernel) bool {
	return true
}

func (k *kernelNaN) divide(other kernel) float64 {
	return math.NaN()
}

type kernelInt64 struct {
	value int64
}

func newKernelInt64(val int64) kernel {
	return &kernelInt64{value: val}
}

func (k *kernelInt64) canAdd(other kernel) bool {
	return other.toInt64() <= math.MaxInt64-k.value
}

func (k *kernelInt64) addInPlace(other kernel) {
	k.value += other.toInt64()
}

func (k *kernelInt64) isNaN() bool {
	return false
}

func (k *kernelInt64) toBigInt() *big.Int {
	return big.NewInt(k.value)
}

func (k *kernelInt64) toInt64() int64 {
	return k.value
}

func (k *kernelInt64) toFloat64() float64 {
	return float64(k.value)
}

func (k *kernelInt64) clone() kernel {
	return newKernelInt64(k.value)
}

func (k *kernelInt64) isIn64BitRange() bool {
	return true
}

func (k *kernelInt64) canDivide(other kernel) bool {
	return other.isIn64BitRange()
}

func (k *kernelInt64) divide(other kernel) float64 {
	return k.toFloat64() / other.toFloat64()
}

type kernelBigInt struct {
	value *big.Int
}

func newKernelBigInt(val *big.Int) kernel {
	return &kernelBigInt{value: val}
}

func (k *kernelBigInt) canAdd(other kernel) bool {
	return true
}

func (k *kernelBigInt) addInPlace(other kernel) {
	k.value.Add(other.toBigInt(), big.NewInt(0))
}

func (k *kernelBigInt) isNaN() bool {
	return false
}

func (k *kernelBigInt) toBigInt() *big.Int {
	return k.value
}

func (k *kernelBigInt) toInt64() int64 {
	if k.value.IsInt64() {
		return k.value.Int64()
	}
	if k.value.Sign() > 0 {
		return math.MaxInt64
	}
	return math.MinInt64
}

func (k *kernelBigInt) toFloat64() float64 {
	if k.value.IsInt64() {
		return float64(k.value.Int64())
	}
	return math.Inf(k.value.Sign())
}

func (k *kernelBigInt) clone() kernel {
	return newKernelBigInt(k.value)
}

func (k *kernelBigInt) isIn64BitRange() bool {
	return k.value.IsInt64()
}

func (k *kernelBigInt) canDivide(other kernel) bool {
	return true
}

func (k *kernelBigInt) divide(other kernel) float64 {
	kFLoat := new(big.Float).SetInt(k.toBigInt())
	otherFloat := new(big.Float).SetInt(other.toBigInt())
	div := new(big.Float).Quo(kFLoat, otherFloat)
	res, acc := div.Float64()

	if acc == big.Above {
		return math.Inf(1)
	}

	if acc == big.Below {
		return math.Inf(-1)
	}

	return res
}
