package loom

import "fmt"

func PairPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(*Pair)
	return Boolean(ok)
}

func PairCons(args Vector) Value {
	if len(args) != 2 {
		panic("cons expects two arguments")
	}
	return Cons(args[0], args[1])
}

func PairCar(args Vector) Value {
	if len(args) != 1 {
		panic("car expects one argument")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("car expects a list")
	}
	return p.Car()
}

func PairCdr(args Vector) Value {
	if len(args) != 1 {
		panic("cdr expects one argument")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("cdr expects a list")
	}
	return p.Cdr()
}

func PairSetCar(args Vector) Value {
	if len(args) != 2 {
		panic("set-car! expects two arguments")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("set-car! expects a list")
	}
	p.car = args[1]
	return nil
}

func PairSetCdr(args Vector) Value {
	if len(args) != 2 {
		panic("set-cdr! expects two arguments")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("set-cdr! expects a list")
	}
	p.cdr = args[1]
	return nil
}

func NullPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	return Boolean(args[0] == nil)
}

func ListConstructor(args Vector) Value {
	return args.ToList()
}

func ListLength(args Vector) Value {
	if len(args) != 1 {
		panic("length expects one argument")
	}
	if args[0] == nil {
		return NewInt(0)
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("length expects a list")
	}
	len := 0
	for p != nil {
		p, _ = p.cdr.(*Pair)
		len++
	}
	return NewInt(int64(len))
}

func ListAppend(args Vector) Value {
	if len(args) == 0 {
		return nil
	}

	var head, tail *Pair
	for _, arg := range args[:len(args)-1] {
		if arg == nil {
			continue
		}

		p, ok := arg.(*Pair)
		if !ok {
			panic("arguments to append must be lists")
		}
		for p != nil {
			e := &Pair{car: p.car}
			if head == nil {
				head, tail = e, e
			} else {
				tail.cdr, tail = e, e
			}
			p, _ = p.cdr.(*Pair)
		}
	}

	if head == nil {
		return args[len(args)-1]
	}
	tail.cdr = args[len(args)-1]
	return head
}

func ListAssq(args Vector) Value {
	if len(args) != 2 {
		panic("assq expects two arguments")
	}

	l, ok := args[1].(*Pair)
	if !ok && args[1] != nil {
		panic("the second argument to assq must be a list of pairs")
	}

	for l != nil {
		p, ok := l.car.(*Pair)
		if !ok {
			panic("the second argument to assq must be a list of pairs")
		}
		if eq(args[0], p.car) {
			return p
		}
		l, _ = l.cdr.(*Pair)
	}

	return Boolean(false)
}

func ListTail(args Vector) Value {
	if len(args) != 2 {
		panic("list-tail expects two arguments")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("the first argument to list-tail must be a list")
	}
	n, ok := args[1].(Number)
	if !ok {
		panic("the second argument to list-tail must be a non-negative integer")
	}
	i, ok := n.Int()
	if !ok || i < 0 {
		panic("the second argument to list-tail must be a non-negative integer")
	}

	for j := i; p != nil && j > 0; j-- {
		p, _ = p.cdr.(*Pair)
	}

	if p == nil {
		panic(fmt.Sprintf("list does not contain %v elements", i))
	}
	return p
}

func ListRef(args Vector) Value {
	if len(args) != 2 {
		panic("list-ref expects two arguments")
	}
	p, ok := args[0].(*Pair)
	if !ok {
		panic("the first argument to list-ref must be a list")
	}
	n, ok := args[1].(Number)
	if !ok {
		panic("the second argument to list-ref must be a non-negative integer")
	}
	i, ok := n.Int()
	if !ok || i < 0 {
		panic("the second argument to list-ref must be a non-negative integer")
	}

	for j := i; p != nil && j > 0; j-- {
		p, _ = p.cdr.(*Pair)
	}

	if p == nil {
		panic(fmt.Sprintf("list does not contain %v elements", i))
	}
	return p.car
}
