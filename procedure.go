package loom

func ProcedurePred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(Procedure)
	return Boolean(ok)
}

func ProcedureApply(args Vector) Value {
	if len(args) < 1 {
		panic("apply expects at least one argument")
	}
	proc, ok := args[0].(Procedure)
	if !ok {
		panic("the first argument to apply must be a procedure")
	}

	var actuals Vector
	if len(args) > 1 {
		l := ListAppend(Vector{ListConstructor(args[1 : len(args)-1]), args[len(args)-1]})
		if l != nil {
			actuals = l.(*Pair).ToVector()
		}
	}

	return proc.Apply(actuals)
}

func ProcedureMap(args Vector) Value {
	if len(args) < 2 {
		panic("map expects at least 2 arguments")
	}

	proc, ok := args[0].(Procedure)
	if !ok {
		panic("the first argument to map must be a procedure")
	}

	lists := args[1:]
	actuals := make(Vector, len(lists))

	var head, tail *Pair
	for {
		for i, arg := range lists {
			l, _ := arg.(*Pair)
			if l == nil {
				return head
			}

			actuals[i], lists[i] = l.car, l.cdr
		}

		p := &Pair{car: proc.Apply(actuals)}
		if head == nil {
			head, tail = p, p
		} else {
			tail.cdr, tail = p, p
		}
	}
}
