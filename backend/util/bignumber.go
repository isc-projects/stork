package storkutil

import (
	"math"
	"math/big"
)

type BigNumber interface {
	Add(other BigNumber) BigNumber
	AddInt64(val int64) BigNumber
	Divide(other BigNumber) float64
	DivideSafe(other BigNumber) float64

	ToInt64() int64
	ToInt64OrDefault(nan int64, above int64) int64

	isNaN() bool

	toBigInt() *big.Int
	toFloat64() float64
}

func NewBigNumber() BigNumber {
	return newBigNumberInt64(0)
}

type BigNumberNaN struct{}

func NewBigNumberNaN() BigNumber {
	return &BigNumberNaN{}
}

func (k *BigNumberNaN) isNaN() bool {
	return true
}
func (k *BigNumberNaN) Add(other BigNumber) BigNumber {
	return k
}
func (k *BigNumberNaN) AddInt64(val int64) BigNumber {
	return k
}
func (k *BigNumberNaN) Divide(other BigNumber) float64 {
	return math.NaN()
}
func (k *BigNumberNaN) DivideSafe(other BigNumber) float64 {
	return math.NaN()
}
func (k *BigNumberNaN) ToInt64() int64 {
	return 0
}
func (k *BigNumberNaN) toBigInt() *big.Int {
	return big.NewInt(0)
}
func (k *BigNumberNaN) ToInt64OrDefault(nan int64, above int64) int64 {
	return nan
}
func (k *BigNumberNaN) toFloat64() float64 {
	return math.NaN()
}

type BigNumberInt64 struct {
	value int64
}

func newBigNumberInt64(value int64) BigNumber {
	return &BigNumberInt64{value}
}

func (k *BigNumberInt64) isNaN() bool {
	return false
}

func (k *BigNumberInt64) Add(other BigNumber) BigNumber {
	if other.isNaN() {
		return NewBigNumberNaN()
	}

	otherValue := other.ToInt64()
	if k.isSumAboveRange(otherValue) {
		kBig := newBigNumberBigInt(k.toBigInt())
		return kBig.Add(other)
	}

	return newBigNumberInt64(k.value + otherValue)
}

func (k *BigNumberInt64) AddInt64(val int64) BigNumber {
	if k.isSumAboveRange(val) {
		kBig := newBigNumberBigInt(k.toBigInt())
		return kBig.AddInt64(val)
	}

	return newBigNumberInt64(k.value + val)
}

func (k *BigNumberInt64) isSumAboveRange(val int64) bool {
	return val > math.MaxInt64-k.value
}

func (k *BigNumberInt64) Divide(other BigNumber) float64 {
	thisValue := k.toFloat64()
	otherValue := other.toFloat64()

	return thisValue / otherValue
}

func (k *BigNumberInt64) DivideSafe(other BigNumber) float64 {
	if other.isNaN() {
		return 0
	}

	val := other.ToInt64()
	if val == 0 {
		return 0
	}

	return k.Divide(other)
}

func (k *BigNumberInt64) ToInt64() int64 {
	return k.value
}

func (k *BigNumberInt64) ToInt64OrDefault(nan int64, above int64) int64 {
	return k.ToInt64()
}

func (k *BigNumberInt64) toBigInt() *big.Int {
	return big.NewInt(k.value)
}

func (k *BigNumberInt64) toFloat64() float64 {
	return float64(k.value)
}

type BigNumberBigInt struct {
	value *big.Int
}

func newBigNumberBigInt(val *big.Int) BigNumber {
	return &BigNumberBigInt{
		value: val,
	}
}

func (k *BigNumberBigInt) isAbove64BitRange() bool {
	return !k.value.IsInt64()
}

func (k *BigNumberBigInt) isNaN() bool {
	return false
}

func (k *BigNumberBigInt) Add(other BigNumber) BigNumber {
	if other.isNaN() {
		return NewBigNumberNaN()
	}
	bigInt := big.NewInt(0).Add(k.value, other.toBigInt())
	return newBigNumberBigInt(bigInt)
}

func (k *BigNumberBigInt) AddInt64(val int64) BigNumber {
	bigInt := big.NewInt(0).Add(k.value, big.NewInt(val))
	return newBigNumberBigInt(bigInt)
}

func (k *BigNumberBigInt) Divide(other BigNumber) float64 {
	return k.toFloat64() / other.toFloat64()
}
func (k *BigNumberBigInt) DivideSafe(other BigNumber) float64 {
	if !other.isNaN() && other.ToInt64() == 0 {
		return 0
	}

	return k.Divide(other)
}

func (k *BigNumberBigInt) ToInt64() int64 {
	if k.isAbove64BitRange() {
		return math.MaxInt64
	}
	return k.value.Int64()
}
func (k *BigNumberBigInt) toBigInt() *big.Int {
	return k.value
}
func (k *BigNumberBigInt) ToInt64OrDefault(nan int64, above int64) int64 {
	if k.isAbove64BitRange() {
		return above
	}
	return k.value.Int64()
}
func (k *BigNumberBigInt) toFloat64() float64 {
	if k.isAbove64BitRange() {
		return math.Inf(1)
	}
	return float64(k.value.Int64())
}
