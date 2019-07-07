package wallet_interface

import (
	"fmt"
	"math/big"
)

// Amount represents the base monetary unit of a currency. For Bitcoin
// this would be the satoshi. For USD this would be the cent. A big.Int
// is used to ensure there is enough room for currencies with large base
// units like Ethereum.
type Amount big.Int

// NewAmount creates an Amount from an interface. The interface can be
// either a int, int32, int64, uint32, uint64, string, or big.Int. Anything
// else will panic.
func NewAmount(i interface{}) Amount {
	switch i.(type) {
	case int:
		return Amount(*big.NewInt(int64(i.(int))))
	case int32:
		return Amount(*big.NewInt(int64(i.(int32))))
	case int64:
		return Amount(*big.NewInt(i.(int64)))
	case uint32:
		return Amount(*big.NewInt(int64(i.(uint32))))
	case uint64:
		a := new(big.Int).SetUint64(i.(uint64))
		return Amount(*a)
	case string:
		a, _ := new(big.Int).SetString(i.(string), 10)
		return Amount(*a)
	case *big.Int:
		a := i.(*big.Int)
		return Amount(*a)
	case big.Int:
		return Amount(i.(big.Int))
	default:
		panic(fmt.Errorf("cannot convert %T to Amount", i))
	}
}

func (a Amount) String() string {
	x := big.Int(a)
	return x.String()
}

func (a Amount) Cmp(b Amount) int {
	x := big.Int(a)
	y := big.Int(b)
	return x.Cmp(&y)
}

func (a Amount) Add(b Amount) Amount {
	x := big.Int(a)
	y := big.Int(b)
	z := new(big.Int).Add(&x, &y)
	return NewAmount(z)
}

func (a Amount) Sub(b Amount) Amount {
	x := big.Int(a)
	y := big.Int(b)
	z := new(big.Int).Sub(&x, &y)
	return NewAmount(z)
}

func (a Amount) Mul(b Amount) Amount {
	x := big.Int(a)
	y := big.Int(b)
	z := new(big.Int).Mul(&x, &y)
	return NewAmount(z)
}

func (a Amount) Div(b Amount) Amount {
	x := big.Int(a)
	y := big.Int(b)
	fx := new(big.Float).SetInt(&x)
	fy := new(big.Float).SetInt(&y)
	fz := new(big.Float).Quo(fx, fy)
	z, _ := fz.Int(nil)
	return NewAmount(z)
}
