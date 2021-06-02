package loom

func BooleanPred(args Vector) Value {
	if len(args) != 1 {
		return Boolean(false)
	}
	_, ok := args[0].(Boolean)
	return Boolean(ok)
}

func BooleanNot(args Vector) Value {
	if len(args) != 1 {
		panic("not expects 1 argument")
	}
	test, ok := args[0].(Boolean)
	return Boolean(ok && !bool(test))
}
