package loom

import "math/big"

func NumberPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(Number)
	return Boolean(ok)
}

func NumberEq(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	n, ok := args[0].(Number)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		if !eqv(n, v) {
			return Boolean(false)
		}
	}

	return Boolean(true)
}

func NumberLt(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	n, ok := args[0].(Number)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok || n.f.Cmp(x.f) != -1 {
			return Boolean(false)
		}
		n = x
	}

	return Boolean(true)
}

func NumberGt(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	n, ok := args[0].(Number)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok || n.f.Cmp(x.f) != 1 {
			return Boolean(false)
		}
		n = x
	}

	return Boolean(true)
}

func NumberLte(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	n, ok := args[0].(Number)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok || n.f.Cmp(x.f) == 1 {
			return Boolean(false)
		}
		n = x
	}

	return Boolean(true)
}

func NumberGte(args Vector) Value {
	if len(args) == 0 {
		return Boolean(true)
	}

	n, ok := args[0].(Number)
	if !ok {
		return Boolean(false)
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok || n.f.Cmp(x.f) == -1 {
			return Boolean(false)
		}
		n = x
	}

	return Boolean(true)
}

func NumberAdd(args Vector) Value {
	if len(args) == 0 {
		return nil
	}

	n, ok := args[0].(Number)
	if !ok {
		return nil
	}

	var sum big.Float
	sum.Copy(n.f)
	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok {
			return nil
		}
		sum.Add(&sum, x.f)
	}
	return Number{f: &sum}
}

func NumberMul(args Vector) Value {
	if len(args) == 0 {
		return nil
	}

	n, ok := args[0].(Number)
	if !ok {
		return nil
	}

	var product big.Float
	product.Copy(n.f)
	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok {
			return nil
		}
		product.Mul(&product, x.f)
	}
	return Number{f: &product}
}

func NumberSub(args Vector) Value {
	if len(args) == 0 {
		return nil
	}

	n, ok := args[0].(Number)
	if !ok {
		return nil
	}
	var diff big.Float
	diff.Copy(n.f)

	if len(args) == 1 {
		diff.Neg(&diff)
		return Number{&diff}
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok {
			return nil
		}
		diff.Sub(&diff, x.f)
	}
	return Number{&diff}
}

func NumberDiv(args Vector) Value {
	if len(args) == 0 {
		return nil
	}

	n, ok := args[0].(Number)
	if !ok {
		return nil
	}
	var quo big.Float
	quo.Copy(n.f)

	if len(args) == 1 {
		quo.Quo(big.NewFloat(1), &quo)
		return Number{&quo}
	}

	for _, v := range args[1:] {
		x, ok := v.(Number)
		if !ok {
			return nil
		}
		quo.Quo(&quo, x.f)
	}
	return Number{&quo}
}

func NumberTruncateQuotient(args Vector) Value {
	if len(args) != 2 {
		panic("truncate-quotient expects 2 arguments")
	}

	n1, ok := args[0].(Number)
	if !ok {
		panic("the first argument to truncate-quotient must be a number")
	}
	n2, ok := args[1].(Number)
	if !ok {
		panic("the second argument to truncate-quotient must be a number")
	}

	var quo big.Float
	quo.Quo(n1.f, n2.f)
	quo.SetMode(big.ToZero)
	i, _ := quo.Int64()
	return NewInt(i)
}
