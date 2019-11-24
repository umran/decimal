package decimal

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

var (
	zeroInt = big.NewInt(0)
	oneInt  = big.NewInt(1)
	fiveInt = big.NewInt(5)
	tenInt  = big.NewInt(10)
)

// Decimal ...
type Decimal struct {
	value *big.Int
	exp   int32
}

// New ...
func New(value int64, exp int32) *Decimal {
	return &Decimal{
		value: big.NewInt(value),
		exp:   exp,
	}
}

// Add ...
func (d *Decimal) Add(d2 *Decimal) *Decimal {
	baseScale := min(d.exp, d2.exp)
	rd := d.rescale(baseScale)
	rd2 := d2.rescale(baseScale)

	d3Value := new(big.Int).Add(rd.value, rd2.value)
	return &Decimal{
		value: d3Value,
		exp:   baseScale,
	}
}

// Sub ...
func (d *Decimal) Sub(d2 *Decimal) *Decimal {
	baseScale := min(d.exp, d2.exp)
	rd := d.rescale(baseScale)
	rd2 := d2.rescale(baseScale)

	d3Value := new(big.Int).Sub(rd.value, rd2.value)
	return &Decimal{
		value: d3Value,
		exp:   baseScale,
	}
}

// Mul ...
func (d *Decimal) Mul(d2 *Decimal) *Decimal {
	expInt64 := int64(d.exp) + int64(d2.exp)
	if expInt64 > math.MaxInt32 || expInt64 < math.MinInt32 {
		// better to panic than give incorrect results, as
		// Decimals are usually used for money
		panic(fmt.Sprintf("exponent %v overflows an int32!", expInt64))
	}

	d3Value := new(big.Int).Mul(d.value, d2.value)
	return &Decimal{
		value: d3Value,
		exp:   int32(expInt64),
	}
}

// StringFixedBank ...
func (d *Decimal) StringFixedBank(places int32) string {
	rounded := d.RoundBank(places)
	return rounded.string(false)
}

// RoundBank ...
func (d *Decimal) RoundBank(places int32) *Decimal {

	round := d.Round(places)
	remainder := d.Sub(round).Abs()

	half := New(5, -places-1)
	if remainder.Cmp(half) == 0 && round.value.Bit(0) != 0 {
		if round.value.Sign() < 0 {
			round.value.Add(round.value, oneInt)
		} else {
			round.value.Sub(round.value, oneInt)
		}
	}

	return round
}

// Round ...
func (d *Decimal) Round(places int32) *Decimal {
	// truncate to places + 1
	ret := d.rescale(-places - 1)

	// add sign(d) * 0.5
	if ret.value.Sign() < 0 {
		ret.value.Sub(ret.value, fiveInt)
	} else {
		ret.value.Add(ret.value, fiveInt)
	}

	// floor for positive numbers, ceil for negative numbers
	_, m := ret.value.DivMod(ret.value, tenInt, new(big.Int))
	ret.exp++
	if ret.value.Sign() < 0 && m.Cmp(zeroInt) != 0 {
		ret.value.Add(ret.value, oneInt)
	}

	return ret
}

// Abs ...
func (d *Decimal) Abs() *Decimal {
	d2Value := new(big.Int).Abs(d.value)
	return &Decimal{
		value: d2Value,
		exp:   d.exp,
	}
}

// Cmp ...
func (d *Decimal) Cmp(d2 *Decimal) int {
	if d.exp == d2.exp {
		return d.value.Cmp(d2.value)
	}

	baseExp := min(d.exp, d2.exp)
	rd := d.rescale(baseExp)
	rd2 := d2.rescale(baseExp)

	return rd.value.Cmp(rd2.value)
}

func (d *Decimal) rescale(exp int32) *Decimal {
	// must convert exps to float64 before - to prevent overflow
	diff := math.Abs(float64(exp) - float64(d.exp))
	value := new(big.Int).Set(d.value)

	expScale := new(big.Int).Exp(tenInt, big.NewInt(int64(diff)), nil)
	if exp > d.exp {
		value = value.Quo(value, expScale)
	} else if exp < d.exp {
		value = value.Mul(value, expScale)
	}

	return &Decimal{
		value: value,
		exp:   exp,
	}
}

func (d Decimal) string(trimTrailingZeros bool) string {
	if d.exp >= 0 {
		return d.rescale(0).value.String()
	}

	abs := new(big.Int).Abs(d.value)
	str := abs.String()

	var intPart, fractionalPart string

	// this cast to int will cause bugs if d.exp == INT_MIN
	// and you are on a 32-bit machine. Won't fix this super-edge case.
	dExpInt := int(d.exp)
	if len(str) > -dExpInt {
		intPart = str[:len(str)+dExpInt]
		fractionalPart = str[len(str)+dExpInt:]
	} else {
		intPart = "0"

		num0s := -dExpInt - len(str)
		fractionalPart = strings.Repeat("0", num0s) + str
	}

	if trimTrailingZeros {
		i := len(fractionalPart) - 1
		for ; i >= 0; i-- {
			if fractionalPart[i] != '0' {
				break
			}
		}
		fractionalPart = fractionalPart[:i+1]
	}

	number := intPart
	if len(fractionalPart) > 0 {
		number += "." + fractionalPart
	}

	if d.value.Sign() < 0 {
		return "-" + number
	}

	return number
}

func min(x, y int32) int32 {
	if x >= y {
		return y
	}
	return x
}
