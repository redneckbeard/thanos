package stdlib

import (
	"fmt"
	"math/big"
)

// Rational represents a Ruby Rational number as an exact fraction.
type Rational struct {
	rat *big.Rat
}

func NewRational(num, denom int64) *Rational {
	return &Rational{rat: new(big.Rat).SetFrac64(num, denom)}
}

func NewRationalFromInt(num int64) *Rational {
	return &Rational{rat: new(big.Rat).SetInt64(num)}
}

func (r *Rational) Numerator() int {
	return int(r.rat.Num().Int64())
}

func (r *Rational) Denominator() int {
	return int(r.rat.Denom().Int64())
}

func (r *Rational) ToF() float64 {
	f, _ := r.rat.Float64()
	return f
}

func (r *Rational) ToI() int {
	// Truncates like Ruby
	return int(r.rat.Num().Int64() / r.rat.Denom().Int64())
}

func (r *Rational) ToS() string {
	return fmt.Sprintf("%d/%d", r.rat.Num().Int64(), r.rat.Denom().Int64())
}

func (r *Rational) String() string {
	return r.ToS()
}

func (r *Rational) Add(other *Rational) *Rational {
	return &Rational{rat: new(big.Rat).Add(r.rat, other.rat)}
}

func (r *Rational) Sub(other *Rational) *Rational {
	return &Rational{rat: new(big.Rat).Sub(r.rat, other.rat)}
}

func (r *Rational) Mul(other *Rational) *Rational {
	return &Rational{rat: new(big.Rat).Mul(r.rat, other.rat)}
}

func (r *Rational) Div(other *Rational) *Rational {
	return &Rational{rat: new(big.Rat).Quo(r.rat, other.rat)}
}

func (r *Rational) Abs() *Rational {
	result := new(big.Rat).Abs(r.rat)
	return &Rational{rat: result}
}

func (r *Rational) Neg() *Rational {
	result := new(big.Rat).Neg(r.rat)
	return &Rational{rat: result}
}
