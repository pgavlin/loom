package loom

// Eqv defines a useful equivalence relation on objects. Briefly, it returns #t
// if obj1 and obj2 are normally regarded as the same object.
func Eqv(args Vector) Value {
	if len(args) != 2 {
		panic("eqv? expects 2 arguments")
	}

	return Boolean(eqv(args[0], args[1]))
}

func eqv(obj1, obj2 Value) bool {
	// The only type we need to treat specially is the Number type. All other
	// types already obey the spec when compared for equality.
	if num1, ok := obj1.(Number); ok {
		num2, ok := obj2.(Number)
		return ok && num1.f.Cmp(num2.f) == 0
	}

	return obj1 == obj2
}

func eq(obj1, obj2 Value) bool {
	return eqv(obj1, obj2)
}

func Eq(args Vector) Value {
	if len(args) != 2 {
		panic("eq? expects 2 arguments")
	}
	return Boolean(eq(args[0], args[1]))
}

func Equal(args Vector) Value {
	if len(args) != 2 {
		panic("equal? expects 2 arguments")
	}

	return Boolean(equal(args[0], args[1], map[Value]struct{}{}))
}

func equal(obj1, obj2 Value, stack map[Value]struct{}) bool {
	if eqv(obj1, obj2) {
		return true
	}

	// TODO: handle self-referential data

	switch obj1 := obj1.(type) {
	case *Pair:
		obj2, ok := obj2.(*Pair)
		if !ok {
			return false
		}

		if _, ok := stack[obj1]; ok {
			return false
		}
		if _, ok := stack[obj2]; ok {
			return false
		}
		stack[obj1], stack[obj2] = struct{}{}, struct{}{}
		defer delete(stack, obj1)
		defer delete(stack, obj2)

		return equal(obj1.car, obj2.car, stack) && equal(obj1.cdr, obj2.cdr, stack)
	case Vector:
		obj2, ok := obj2.(Vector)
		if !ok {
			return false
		}

		if len(obj1) != len(obj2) {
			return false
		}

		if _, ok := stack[obj1]; ok {
			return false
		}
		if _, ok := stack[obj2]; ok {
			return false
		}
		stack[obj1], stack[obj2] = struct{}{}, struct{}{}
		defer delete(stack, obj1)
		defer delete(stack, obj2)

		for i, e := range obj1 {
			if !equal(e, obj2[i], stack) {
				return false
			}
		}

		return true
	default:
		return false
	}
}
