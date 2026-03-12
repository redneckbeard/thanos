package stdlib

import (
	"math"
	"math/big"
)

// Abs returns the absolute value of an integer.
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// NewRationalFromFloat converts a float64 to a Rational.
func NewRationalFromFloat(f float64) *Rational {
	r := new(big.Rat).SetFloat64(f)
	return &Rational{rat: r}
}

// FloatDivmod returns [quotient, modulus] like Ruby's Float#divmod.
func FloatDivmod(a, b float64) []float64 {
	q := math.Floor(a / b)
	m := a - q*b
	return []float64{q, m}
}
